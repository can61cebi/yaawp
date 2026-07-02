package engine

import (
	"log"
	"strings"

	"cebi.tr/yaawp/internal/ipc"

	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/proto/waWeb"
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
		msg := messageToIPC(evt)
		if err := e.db.PutMessage(msg); err != nil {
			log.Printf("persist message: %v", err)
		}
		e.emit(ipc.NewEvent(ipc.EventMessage, msg))
	case *events.Receipt:
		e.emit(ipc.NewEvent(ipc.EventReceipt, map[string]any{
			"chat_jid":    evt.Chat.String(),
			"message_ids": evt.MessageIDs,
			"receipt":     string(evt.Type),
		}))
	case *events.Presence:
		state := "available"
		if evt.Unavailable {
			state = "unavailable"
		}
		e.emit(ipc.NewEvent(ipc.EventPresence, map[string]any{
			"jid":   evt.From.String(),
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
		chatJID := conv.GetID()
		if chatJID == "" {
			continue
		}
		if name := conv.GetName(); name != "" {
			_ = e.db.SetChatName(chatJID, name, strings.HasSuffix(chatJID, "@g.us"))
		}
		batch := make([]ipc.Message, 0, len(conv.GetMessages()))
		for _, hm := range conv.GetMessages() {
			wmi := hm.GetMessage()
			if wmi == nil {
				continue
			}
			m := webMsgToIPC(chatJID, wmi)
			if m.ID == "" {
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

func messageToIPC(evt *events.Message) ipc.Message {
	return ipc.Message{
		ID:        evt.Info.ID,
		ChatJID:   evt.Info.Chat.String(),
		SenderJID: evt.Info.Sender.String(),
		FromMe:    evt.Info.IsFromMe,
		Timestamp: evt.Info.Timestamp.Unix(),
		Type:      "text",
		Text:      extractTextFromMessage(evt.Message),
	}
}

func webMsgToIPC(chatJID string, wmi *waWeb.WebMessageInfo) ipc.Message {
	key := wmi.GetKey()
	return ipc.Message{
		ID:        key.GetID(),
		ChatJID:   chatJID,
		SenderJID: senderFromKey(chatJID, key),
		FromMe:    key.GetFromMe(),
		Timestamp: int64(wmi.GetMessageTimestamp()),
		Type:      "text",
		Text:      extractTextFromMessage(wmi.GetMessage()),
	}
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

// extractTextFromMessage returns the textual content of a message, if any.
func extractTextFromMessage(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if c := msg.GetConversation(); c != "" {
		return c
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}
