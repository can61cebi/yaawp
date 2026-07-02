package engine

import (
	"log"

	"cebi.tr/yaawp/internal/ipc"

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
		// TODO: persist the conversations in evt.Data into the local store.
		e.emit(ipc.NewEvent(ipc.EventHistorySync, map[string]any{
			"progress": evt.Data.GetProgress(),
		}))
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
		Text:      extractText(evt),
	}
}

// extractText returns the textual content of a message, if any.
func extractText(evt *events.Message) string {
	if evt.Message == nil {
		return ""
	}
	if c := evt.Message.GetConversation(); c != "" {
		return c
	}
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}
