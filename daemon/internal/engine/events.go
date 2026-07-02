package engine

import (
	"log"
	"strings"

	"cebi.tr/yaawp/internal/ipc"

	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// handleEvent translates whatsmeow events into IPC events, persists what is
// relevant, and forwards them to connected clients.
func (e *Engine) handleEvent(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Connected:
		e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "connected"}))
	case *events.Disconnected:
		e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "disconnected"}))
	case *events.LoggedOut:
		e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "logged_out"}))
	case *events.PairSuccess:
		e.emit(ipc.NewEvent(ipc.EventPairSuccess, map[string]any{
			"jid":       evt.ID.String(),
			"push_name": e.client.Store.PushName,
		}))
	case *events.Message:
		msg, ok := e.messageToIPC(evt)
		if !ok {
			return // non-renderable (protocol, reaction, empty)
		}
		if err := e.db.PutMessage(msg); err != nil {
			log.Printf("persist message: %v", err)
		}
		e.emit(ipc.NewEvent(ipc.EventMessage, msg))
	case *events.Receipt:
		e.emit(ipc.NewEvent(ipc.EventReceipt, map[string]any{
			"chat_jid":    e.canonicalJID(evt.Chat.String()),
			"message_ids": evt.MessageIDs,
			"receipt":     string(evt.Type),
		}))
	case *events.Presence:
		state := "available"
		if evt.Unavailable {
			state = "unavailable"
		}
		e.emit(ipc.NewEvent(ipc.EventPresence, map[string]any{
			"jid":   e.canonicalJID(evt.From.String()),
			"state": state,
		}))
	case *events.HistorySync:
		e.persistHistory(evt.Data)
		e.emit(ipc.NewEvent(ipc.EventHistorySync, map[string]any{
			"progress": evt.Data.GetProgress(),
		}))
	}
}

// persistHistory stores the conversations and messages from a history sync
// payload, one transaction per conversation.
func (e *Engine) persistHistory(data *waHistorySync.HistorySync) {
	if data == nil {
		return
	}
	total := 0
	for _, conv := range data.GetConversations() {
		rawID := conv.GetID()
		if rawID == "" {
			continue
		}
		chatJID := e.canonicalJID(rawID)
		if name := conv.GetName(); name != "" {
			_ = e.db.SetChatName(chatJID, name, strings.HasSuffix(chatJID, "@"+types.GroupServer))
		}
		batch := make([]ipc.Message, 0, len(conv.GetMessages()))
		for _, hm := range conv.GetMessages() {
			wmi := hm.GetMessage()
			if wmi == nil {
				continue
			}
			m, ok := e.webMsgToIPC(chatJID, wmi)
			if !ok || m.ID == "" {
				continue
			}
			batch = append(batch, m)
		}
		if err := e.db.PutMessages(batch); err != nil {
			log.Printf("persist history for %s: %v", chatJID, err)
			continue
		}
		total += len(batch)
	}
	if total > 0 {
		log.Printf("history sync: stored %d messages", total)
	}
}

// messageToIPC converts a live message event. It returns ok=false for messages
// that have no user-visible content (protocol, reaction, and similar).
func (e *Engine) messageToIPC(evt *events.Message) (ipc.Message, bool) {
	typ, text, ok := describeMessage(evt.Message)
	if !ok {
		return ipc.Message{}, false
	}
	return ipc.Message{
		ID:        evt.Info.ID,
		ChatJID:   e.canonicalJID(evt.Info.Chat.String()),
		SenderJID: e.canonicalJID(evt.Info.Sender.String()),
		FromMe:    evt.Info.IsFromMe,
		Timestamp: evt.Info.Timestamp.Unix(),
		Type:      typ,
		Text:      text,
	}, true
}

// webMsgToIPC converts a history message. chatJID is expected to be canonical.
func (e *Engine) webMsgToIPC(chatJID string, wmi *waWeb.WebMessageInfo) (ipc.Message, bool) {
	typ, text, ok := describeMessage(wmi.GetMessage())
	if !ok {
		return ipc.Message{}, false
	}
	key := wmi.GetKey()
	return ipc.Message{
		ID:        key.GetID(),
		ChatJID:   chatJID,
		SenderJID: e.canonicalJID(senderFromKey(chatJID, key)),
		FromMe:    key.GetFromMe(),
		Timestamp: int64(wmi.GetMessageTimestamp()),
		Type:      typ,
		Text:      text,
	}, true
}

// senderFromKey derives the sender JID from a message key. Group messages carry
// a participant; one-to-one messages use the chat JID.
func senderFromKey(chatJID string, key *waCommon.MessageKey) string {
	if key == nil {
		return chatJID
	}
	if p := key.GetParticipant(); p != "" {
		return p
	}
	if key.GetFromMe() {
		return ""
	}
	return chatJID
}

// describeMessage returns a message type and a display text. ok is false when
// the message has no user-visible content and should be skipped.
func describeMessage(msg *waE2E.Message) (typ, text string, ok bool) {
	if msg == nil {
		return "", "", false
	}
	if c := msg.GetConversation(); c != "" {
		return "text", c, true
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		if t := ext.GetText(); t != "" {
			return "text", t, true
		}
	}
	if img := msg.GetImageMessage(); img != nil {
		return "image", fallbackText(img.GetCaption(), "[image]"), true
	}
	if vid := msg.GetVideoMessage(); vid != nil {
		return "video", fallbackText(vid.GetCaption(), "[video]"), true
	}
	if msg.GetAudioMessage() != nil {
		return "audio", "[voice message]", true
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		name := doc.GetFileName()
		if name == "" {
			name = doc.GetTitle()
		}
		return "document", fallbackText(name, "[document]"), true
	}
	if msg.GetStickerMessage() != nil {
		return "sticker", "[sticker]", true
	}
	if msg.GetContactMessage() != nil {
		return "contact", "[contact]", true
	}
	if msg.GetLocationMessage() != nil {
		return "location", "[location]", true
	}
	return "", "", false
}

func fallbackText(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
