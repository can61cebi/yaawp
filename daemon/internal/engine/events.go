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
	"google.golang.org/protobuf/proto"
)

// handleEvent translates whatsmeow events into IPC events, persists what is
// relevant, and forwards them to connected clients.
func (e *Engine) handleEvent(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.Connected:
		e.setQR("")
		// Announce availability so the server delivers others' presence to us.
		if !e.onlineHidden() {
			go func() { _ = e.client.SendPresence(e.ctx, types.PresenceAvailable) }()
		}
		e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "connected"}))
	case *events.Disconnected:
		e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "disconnected"}))
	case *events.LoggedOut:
		e.setQR("")
		e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "logged_out"}))
	case *events.PairSuccess:
		e.setQR("")
		e.emit(ipc.NewEvent(ipc.EventPairSuccess, map[string]any{
			"jid":       evt.ID.String(),
			"push_name": e.client.Store.PushName,
		}))
	case *events.Message:
		if e.handleSpecialMessage(evt) {
			return
		}
		msg, ok := e.messageToIPC(evt)
		if !ok {
			return // non-renderable (protocol, reaction, empty)
		}
		if err := e.db.PutMessage(msg); err != nil {
			log.Printf("persist message: %v", err)
		}
		e.emit(ipc.NewEvent(ipc.EventMessage, msg))
		e.maybeDownloadMedia(msg.ChatJID, msg.ID, evt.Message)
		e.bumpUnread(msg)
	case *events.Receipt:
		chatJID := e.canonicalJID(evt.Chat.String())
		ids := make([]string, len(evt.MessageIDs))
		for i, id := range evt.MessageIDs {
			ids[i] = string(id)
		}
		status := ""
		switch evt.Type {
		case types.ReceiptTypeDelivered:
			status = "delivered"
		case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
			status = "read"
		}
		if status != "" {
			_ = e.db.UpdateStatus(chatJID, ids, status)
			e.emit(ipc.NewEvent(ipc.EventMessageStatus, map[string]any{
				"chat_jid":    chatJID,
				"message_ids": ids,
				"status":      status,
			}))
		}
		e.emit(ipc.NewEvent(ipc.EventReceipt, map[string]any{
			"chat_jid":    chatJID,
			"message_ids": ids,
			"receipt":     string(evt.Type),
		}))
	case *events.Presence:
		data := map[string]any{"jid": e.canonicalJID(evt.From.String()), "state": "available"}
		if evt.Unavailable {
			data["state"] = "unavailable"
		}
		if !evt.LastSeen.IsZero() {
			data["last_seen"] = evt.LastSeen.Unix()
		}
		e.emit(ipc.NewEvent(ipc.EventPresence, data))
	case *events.ChatPresence:
		e.emit(ipc.NewEvent(ipc.EventChatPresence, map[string]any{
			"chat_jid":   e.canonicalJID(evt.Chat.String()),
			"sender_jid": e.canonicalJID(evt.Sender.String()),
			"state":      string(evt.State),
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

// handleSpecialMessage processes messages that are not plain content, such as
// a revoke. It returns true when the message was fully handled.
func (e *Engine) handleSpecialMessage(evt *events.Message) bool {
	m := evt.Message
	if m == nil {
		return false
	}
	if pm := m.GetProtocolMessage(); pm != nil && pm.GetType() == waE2E.ProtocolMessage_REVOKE {
		chatJID := e.canonicalJID(evt.Info.Chat.String())
		targetID := pm.GetKey().GetID()
		_ = e.db.MarkRevoked(chatJID, targetID)
		e.emit(ipc.NewEvent(ipc.EventMessageRevoked, map[string]any{
			"chat_jid":   chatJID,
			"message_id": targetID,
		}))
		return true
	}
	if pm := m.GetProtocolMessage(); pm != nil && pm.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT {
		chatJID := e.canonicalJID(evt.Info.Chat.String())
		targetID := pm.GetKey().GetID()
		_, newText, ok := describeMessage(pm.GetEditedMessage())
		if !ok {
			return true
		}
		_ = e.db.UpdateText(chatJID, targetID, newText)
		e.emit(ipc.NewEvent(ipc.EventMessageEdited, map[string]any{
			"chat_jid":   chatJID,
			"message_id": targetID,
			"text":       newText,
		}))
		return true
	}
	if r := m.GetReactionMessage(); r != nil {
		chatJID := e.canonicalJID(evt.Info.Chat.String())
		targetID := r.GetKey().GetID()
		senderJID := e.canonicalJID(evt.Info.Sender.String())
		emoji := r.GetText()
		_ = e.db.PutReaction(chatJID, targetID, senderJID, emoji)
		e.emit(ipc.NewEvent(ipc.EventReaction, map[string]any{
			"chat_jid":   chatJID,
			"message_id": targetID,
			"sender_jid": senderJID,
			"emoji":      emoji,
			"from_me":    evt.Info.IsFromMe,
		}))
		return true
	}
	return false
}

// messageToIPC converts a live message event. It returns ok=false for messages
// that have no user-visible content (protocol, reaction, and similar).
func (e *Engine) messageToIPC(evt *events.Message) (ipc.Message, bool) {
	typ, text, ok := describeMessage(evt.Message)
	if !ok {
		return ipc.Message{}, false
	}
	status := ""
	if evt.Info.IsFromMe {
		status = "sent"
	}
	qid, qsender, qtext := e.extractQuote(evt.Message)
	mw, mh := mediaDimensions(evt.Message)
	raw := rawMedia(evt.Message)
	return ipc.Message{
		ID:           evt.Info.ID,
		ChatJID:      e.canonicalJID(evt.Info.Chat.String()),
		SenderJID:    e.canonicalJID(evt.Info.Sender.String()),
		SenderName:   evt.Info.PushName,
		FromMe:       evt.Info.IsFromMe,
		Timestamp:    evt.Info.Timestamp.Unix(),
		Type:         typ,
		Text:         text,
		Status:       status,
		QuotedID:     qid,
		QuotedSender: qsender,
		QuotedText:   qtext,
		MediaWidth:   mw,
		MediaHeight:  mh,
		RawMedia:     raw,
	}, true
}

// extractQuote returns the quoted message id, sender, and text when a message
// is a reply.
func (e *Engine) extractQuote(m *waE2E.Message) (id, sender, text string) {
	if m == nil {
		return "", "", ""
	}
	ext := m.GetExtendedTextMessage()
	if ext == nil {
		return "", "", ""
	}
	ci := ext.GetContextInfo()
	if ci == nil || ci.GetQuotedMessage() == nil {
		return "", "", ""
	}
	_, text, ok := describeMessage(ci.GetQuotedMessage())
	if !ok || text == "" {
		text = "[media]"
	}
	return ci.GetStanzaID(), e.canonicalJID(ci.GetParticipant()), text
}

// webMsgToIPC converts a history message. chatJID is expected to be canonical.
func (e *Engine) webMsgToIPC(chatJID string, wmi *waWeb.WebMessageInfo) (ipc.Message, bool) {
	typ, text, ok := describeMessage(wmi.GetMessage())
	if !ok {
		return ipc.Message{}, false
	}
	key := wmi.GetKey()
	status := ""
	if key.GetFromMe() {
		status = "sent"
	}
	qid, qsender, qtext := e.extractQuote(wmi.GetMessage())
	mw, mh := mediaDimensions(wmi.GetMessage())
	raw := rawMedia(wmi.GetMessage())
	return ipc.Message{
		ID:           key.GetID(),
		ChatJID:      chatJID,
		SenderJID:    e.canonicalJID(senderFromKey(chatJID, key)),
		FromMe:       key.GetFromMe(),
		Timestamp:    int64(wmi.GetMessageTimestamp()),
		Type:         typ,
		Text:         text,
		Status:       status,
		QuotedID:     qid,
		QuotedSender: qsender,
		QuotedText:   qtext,
		MediaWidth:   mw,
		MediaHeight:  mh,
		RawMedia:     raw,
	}, true
}

// mediaDimensions returns the pixel width and height of an image, sticker, or
// video message so the GUI can reserve layout space before the file loads.
func mediaDimensions(m *waE2E.Message) (int, int) {
	if m == nil {
		return 0, 0
	}
	if img := m.GetImageMessage(); img != nil {
		return int(img.GetWidth()), int(img.GetHeight())
	}
	if st := m.GetStickerMessage(); st != nil {
		return int(st.GetWidth()), int(st.GetHeight())
	}
	if vid := m.GetVideoMessage(); vid != nil {
		return int(vid.GetWidth()), int(vid.GetHeight())
	}
	return 0, 0
}

// rawMedia marshals the downloadable submessage of a message so it can be stored
// and later used to fetch the attachment on demand. It returns nil for messages
// without an attachment.
func rawMedia(m *waE2E.Message) []byte {
	if m == nil {
		return nil
	}
	var sub proto.Message
	switch {
	case m.GetImageMessage() != nil:
		sub = m.GetImageMessage()
	case m.GetVideoMessage() != nil:
		sub = m.GetVideoMessage()
	case m.GetAudioMessage() != nil:
		sub = m.GetAudioMessage()
	case m.GetDocumentMessage() != nil:
		sub = m.GetDocumentMessage()
	case m.GetStickerMessage() != nil:
		sub = m.GetStickerMessage()
	default:
		return nil
	}
	data, err := proto.Marshal(sub)
	if err != nil {
		return nil
	}
	return data
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
