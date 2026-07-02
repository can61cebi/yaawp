// Package ipc defines the newline-delimited JSON contract between the daemon
// and the GUI, and the Unix socket server that carries it.
// Schema reference: /ipc/protocol.md
package ipc

import "encoding/json"

// Envelope types.
const (
	TypeCommand  = "cmd"
	TypeResponse = "resp"
	TypeEvent    = "event"
)

// Command method names (GUI to daemon).
const (
	MethodGetState     = "get_state"
	MethodLogin        = "login"
	MethodLogout       = "logout"
	MethodListChats    = "list_chats"
	MethodListMessages = "list_messages"
	MethodSendText          = "send_text"
	MethodMarkRead          = "mark_read"
	MethodSetTyping         = "set_typing"
	MethodSubscribePresence = "subscribe_presence"
	MethodDeleteMessage     = "delete_message"
	MethodSendReaction      = "send_reaction"
	MethodSendMedia         = "send_media"
	MethodDownloadMedia     = "download_media"
)

// Event names (daemon to GUI).
const (
	EventQR          = "qr"
	EventPairSuccess = "pair_success"
	EventConnection  = "connection"
	EventMessage     = "message"
	EventReceipt     = "receipt"
	EventPresence     = "presence"
	EventHistorySync  = "history_sync"
	EventChatPresence  = "chat_presence"
	EventMessageStatus  = "message_status"
	EventMessageMedia   = "message_media"
	EventMessageRevoked = "message_revoked"
	EventReaction       = "reaction"
)

// Command is a request from a GUI client.
type Command struct {
	Type   string          `json:"type"` // always "cmd"
	ID     string          `json:"id"`   // correlation id
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is the reply to a Command.
type Response struct {
	Type   string      `json:"type"` // always "resp"
	ID     string      `json:"id"`
	OK     bool        `json:"ok"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Event is an unsolicited push from the daemon.
type Event struct {
	Type  string      `json:"type"` // always "event"
	Event string      `json:"event"`
	Data  interface{} `json:"data,omitempty"`
}

// NewEvent builds an Event with the given name and data payload.
func NewEvent(name string, data interface{}) Event {
	return Event{Type: TypeEvent, Event: name, Data: data}
}

// ---- Parameter and data payloads ----

type SendTextParams struct {
	ChatJID      string `json:"chat_jid"`
	Text         string `json:"text"`
	QuotedID     string `json:"quoted_id,omitempty"`
	QuotedSender string `json:"quoted_sender,omitempty"`
	QuotedText   string `json:"quoted_text,omitempty"`
}

type ListMessagesParams struct {
	ChatJID string `json:"chat_jid"`
	Limit   int    `json:"limit"`
	Before  string `json:"before,omitempty"`
}

type MarkReadParams struct {
	ChatJID    string   `json:"chat_jid"`
	MessageIDs []string `json:"message_ids"`
}

type SetTypingParams struct {
	ChatJID   string `json:"chat_jid"`
	Composing bool   `json:"composing"`
}

type SubscribePresenceParams struct {
	JID string `json:"jid"`
}

type DeleteMessageParams struct {
	ChatJID   string `json:"chat_jid"`
	MessageID string `json:"message_id"`
}

type SendReactionParams struct {
	ChatJID   string `json:"chat_jid"`
	MessageID string `json:"message_id"`
	SenderJID string `json:"sender_jid"`
	FromMe    bool   `json:"from_me"`
	Emoji     string `json:"emoji"`
}

type SendMediaParams struct {
	ChatJID  string `json:"chat_jid"`
	FilePath string `json:"file_path"`
	Caption  string `json:"caption,omitempty"`
}

type Chat struct {
	JID         string `json:"jid"`
	Name        string `json:"name"`
	IsGroup     bool   `json:"is_group"`
	LastTS      int64  `json:"last_message_ts"`
	LastPreview string `json:"last_message_preview"`
	UnreadCount int    `json:"unread_count"`
}

type Message struct {
	ID         string `json:"id"`
	ChatJID    string `json:"chat_jid"`
	SenderJID  string `json:"sender_jid"`
	SenderName string `json:"sender_name,omitempty"`
	FromMe     bool   `json:"from_me"`
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"` // text, image, audio, and so on
	Text      string `json:"text,omitempty"`
	Status      string `json:"status,omitempty"`     // sent, delivered, read (outgoing only)
	MediaPath   string `json:"media_path,omitempty"` // local cache path for downloaded media
	MediaWidth  int    `json:"media_w,omitempty"`    // media pixel dimensions, to reserve layout space
	MediaHeight int    `json:"media_h,omitempty"`
	RawMedia    []byte `json:"-"` // marshaled protobuf of the media submessage, for on demand download

	Reactions map[string]string `json:"reactions,omitempty"` // sender jid -> emoji

	QuotedID     string `json:"quoted_id,omitempty"`
	QuotedSender string `json:"quoted_sender,omitempty"`
	QuotedText   string `json:"quoted_text,omitempty"`
}

// DownloadMediaParams requests an on demand download of a message's attachment.
type DownloadMediaParams struct {
	ChatJID   string `json:"chat_jid"`
	MessageID string `json:"message_id"`
}

// Backend is implemented by the engine. The IPC server dispatches commands to
// it and returns a JSON-serialisable result or an error.
type Backend interface {
	// InitialEvents returns events for a newly connected client so it can
	// render the current state without waiting for the next push.
	InitialEvents() []Event
	GetState() (interface{}, error)
	Login() (interface{}, error)
	Logout() (interface{}, error)
	ListChats() (interface{}, error)
	ListMessages(p ListMessagesParams) (interface{}, error)
	SendText(p SendTextParams) (interface{}, error)
	MarkRead(p MarkReadParams) (interface{}, error)
	SetTyping(p SetTypingParams) (interface{}, error)
	SubscribePresence(p SubscribePresenceParams) (interface{}, error)
	DeleteMessage(p DeleteMessageParams) (interface{}, error)
	SendReaction(p SendReactionParams) (interface{}, error)
	SendMedia(p SendMediaParams) (interface{}, error)
	DownloadMedia(p DownloadMediaParams) (interface{}, error)
}
