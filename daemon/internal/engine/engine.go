// Package engine wraps the whatsmeow client and implements the ipc.Backend
// interface.
package engine

import (
	"context"
	"fmt"
	"sync"

	"cebi.tr/yaawp/internal/ipc"
	"cebi.tr/yaawp/internal/store"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Engine holds the WhatsApp session and forwards protocol events to the GUI
// through the installed sink. It persists chats and messages in a local store.
type Engine struct {
	ctx    context.Context
	client *whatsmeow.Client
	db     *store.DB

	sinkMu sync.RWMutex
	sink   func(ipc.Event)
}

// New opens the session and application stores, creates the client and attaches
// the event handler.
func New(ctx context.Context) (*Engine, error) {
	dbPath, err := store.DatabasePath()
	if err != nil {
		return nil, err
	}
	dbLog := waLog.Stdout("DB", "INFO", true)
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on", dbPath)
	container, err := sqlstore.New(ctx, "sqlite3", dsn, dbLog)
	if err != nil {
		return nil, fmt.Errorf("open session store: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get device: %w", err)
	}

	appDB, err := store.OpenDB()
	if err != nil {
		return nil, fmt.Errorf("open app store: %w", err)
	}

	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true))

	e := &Engine{ctx: ctx, client: client, db: appDB}
	client.AddEventHandler(e.handleEvent)
	return e, nil
}

// SetSink installs the callback used to push events to IPC clients.
func (e *Engine) SetSink(sink func(ipc.Event)) {
	e.sinkMu.Lock()
	e.sink = sink
	e.sinkMu.Unlock()
}

func (e *Engine) emit(evt ipc.Event) {
	e.sinkMu.RLock()
	sink := e.sink
	e.sinkMu.RUnlock()
	if sink != nil {
		sink(evt)
	}
}

// Start connects to WhatsApp. If the device is not paired yet, it begins QR
// login and streams QR codes as events.
func (e *Engine) Start() error {
	if e.client.Store.ID == nil {
		return e.beginQRLogin()
	}
	return e.client.Connect()
}

// beginQRLogin opens the QR channel, connects, and emits each QR code as an event.
func (e *Engine) beginQRLogin() error {
	qrChan, err := e.client.GetQRChannel(e.ctx)
	if err != nil {
		return err
	}
	if err := e.client.Connect(); err != nil {
		return err
	}
	go func() {
		for item := range qrChan {
			switch item.Event {
			case "code":
				e.emit(ipc.NewEvent(ipc.EventQR, map[string]any{"code": item.Code}))
			case "success":
				// Pairing is done; connection events will follow.
			default:
				e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": item.Event}))
			}
		}
	}()
	return nil
}

// Disconnect closes the connection and the application store cleanly.
func (e *Engine) Disconnect() {
	e.client.Disconnect()
	if e.db != nil {
		_ = e.db.Close()
	}
}

// ---- ipc.Backend implementation ----

func (e *Engine) GetState() (interface{}, error) {
	state := "logged_out"
	var jid string
	if e.client.Store.ID != nil {
		jid = e.client.Store.ID.String()
		if e.client.IsConnected() {
			state = "connected"
		} else {
			state = "disconnected"
		}
	}
	return map[string]any{"state": state, "jid": jid}, nil
}

func (e *Engine) Login() (interface{}, error) {
	if e.client.Store.ID != nil {
		return map[string]any{"already": true}, nil
	}
	if err := e.beginQRLogin(); err != nil {
		return nil, err
	}
	return map[string]any{"started": true}, nil
}

func (e *Engine) Logout() (interface{}, error) {
	if err := e.client.Logout(e.ctx); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

func (e *Engine) ListChats() (interface{}, error) {
	chats, err := e.db.ListChats()
	if err != nil {
		return nil, err
	}
	// Best effort: resolve display names for one-to-one chats from the
	// contact store and cache them for next time.
	for i := range chats {
		if chats[i].Name != "" || chats[i].IsGroup {
			continue
		}
		if name := e.resolveContactName(chats[i].JID); name != "" {
			chats[i].Name = name
			_ = e.db.SetChatName(chats[i].JID, name, false)
		}
	}
	return chats, nil
}

// resolveContactName looks up a cached display name for a user JID.
func (e *Engine) resolveContactName(jidStr string) string {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return ""
	}
	info, err := e.client.Store.Contacts.GetContact(e.ctx, jid)
	if err != nil || !info.Found {
		return ""
	}
	switch {
	case info.FullName != "":
		return info.FullName
	case info.PushName != "":
		return info.PushName
	case info.BusinessName != "":
		return info.BusinessName
	case info.FirstName != "":
		return info.FirstName
	}
	return ""
}

func (e *Engine) ListMessages(p ipc.ListMessagesParams) (interface{}, error) {
	return e.db.ListMessages(p.ChatJID, p.Limit)
}

func (e *Engine) SendText(p ipc.SendTextParams) (interface{}, error) {
	jid, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	msg := &waE2E.Message{Conversation: proto.String(p.Text)}
	resp, err := e.client.SendMessage(e.ctx, jid, msg)
	if err != nil {
		return nil, err
	}
	// Persist the outgoing message so it survives restarts.
	_ = e.db.PutMessage(ipc.Message{
		ID:        resp.ID,
		ChatJID:   p.ChatJID,
		FromMe:    true,
		Timestamp: resp.Timestamp.Unix(),
		Type:      "text",
		Text:      p.Text,
	})
	return map[string]any{"message_id": resp.ID, "timestamp": resp.Timestamp.Unix()}, nil
}

// MarkRead is a placeholder. TODO: call client.MarkRead with the correct
// sender, chat and message ids.
func (e *Engine) MarkRead(p ipc.MarkReadParams) (interface{}, error) {
	return map[string]any{"ok": true}, nil
}
