// Package engine wraps the whatsmeow client and implements the ipc.Backend
// interface.
package engine

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cebi.tr/yaawp/internal/ipc"
	"cebi.tr/yaawp/internal/store"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waE2E"
	waStore "go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// deviceOS is the label shown under WhatsApp Linked Devices. A browser-like
// value blends in with a normal web session instead of exposing the library
// name. Change it to "yaawp" for a visible project label. The name is fixed at
// pair time, so an already linked device keeps whatever name it was paired with.
const deviceOS = "Chrome (Linux)"

// Engine holds the WhatsApp session and forwards protocol events to the GUI
// through the installed sink. It persists chats and messages in a local store.
type Engine struct {
	ctx    context.Context
	client *whatsmeow.Client
	db     *store.DB

	sinkMu sync.RWMutex
	sink   func(ipc.Event)

	qrMu   sync.Mutex
	lastQR string

	activeMu   sync.Mutex
	activeChat string // chat currently on screen; its messages are not unread
}

// New opens the session and application stores, creates the client and attaches
// the event handler.
func New(ctx context.Context) (*Engine, error) {
	// Present a browser-like device identity rather than the library default.
	waStore.DeviceProps.Os = proto.String(deviceOS)
	waStore.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()

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

func (e *Engine) setQR(code string) {
	e.qrMu.Lock()
	e.lastQR = code
	e.qrMu.Unlock()
}

// InitialEvents returns the events a newly connected client should receive so
// it can render the current state immediately: the connection state and, if
// pairing is in progress, the latest QR code. Without this a client that
// connects after the QR was generated would wait for the next refresh.
func (e *Engine) InitialEvents() []ipc.Event {
	state := "logged_out"
	if e.client.Store.ID != nil {
		if e.client.IsConnected() {
			state = "connected"
		} else {
			state = "connecting"
		}
	} else {
		state = "connecting"
	}
	evts := []ipc.Event{ipc.NewEvent(ipc.EventConnection, map[string]any{"state": state})}

	e.qrMu.Lock()
	qr := e.lastQR
	e.qrMu.Unlock()
	if qr != "" && state != "connected" {
		evts = append(evts, ipc.NewEvent(ipc.EventQR, map[string]any{"code": qr}))
	}
	return evts
}

// Start connects to WhatsApp. If the device is not paired yet, it begins QR
// login and streams QR codes as events.
func (e *Engine) Start() error {
	if e.client.Store.ID == nil {
		return e.beginQRLogin()
	}
	e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "connecting"}))
	return e.client.Connect()
}

// beginQRLogin opens the QR channel, connects, and emits each QR code as an event.
func (e *Engine) beginQRLogin() error {
	e.emit(ipc.NewEvent(ipc.EventConnection, map[string]any{"state": "connecting"}))
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
				e.setQR(item.Code)
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

// canonicalJID maps a hidden LID user to its phone-number JID when a mapping is
// known, so a single contact does not appear under two identities.
func (e *Engine) canonicalJID(jidStr string) string {
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return jidStr
	}
	if jid.Server != types.HiddenUserServer {
		return jidStr
	}
	if e.client.Store.LIDs == nil {
		return jidStr
	}
	pn, err := e.client.Store.LIDs.GetPNForLID(e.ctx, jid)
	if err != nil || pn.IsEmpty() {
		return jidStr
	}
	return pn.String()
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

// GroupInfo returns a group's metadata and participant list with resolved names.
func (e *Engine) GroupInfo(p ipc.GroupInfoParams) (interface{}, error) {
	jid, err := types.ParseJID(p.JID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	gi, err := e.client.GetGroupInfo(e.ctx, jid)
	if err != nil {
		return nil, err
	}
	parts := make([]map[string]any, 0, len(gi.Participants))
	for _, pt := range gi.Participants {
		name := e.resolveContactName(pt.JID.String())
		if name == "" {
			name = pt.DisplayName
		}
		if name == "" {
			name = pt.JID.User
		}
		parts = append(parts, map[string]any{
			"jid":      e.canonicalJID(pt.JID.String()),
			"name":     name,
			"is_admin": pt.IsAdmin || pt.IsSuperAdmin,
		})
	}
	return map[string]any{
		"jid":               p.JID,
		"name":              gi.Name,
		"topic":             gi.Topic,
		"participant_count": len(gi.Participants),
		"participants":      parts,
	}, nil
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
	msgs, err := e.db.ListMessages(p.ChatJID, p.Limit)
	if err != nil {
		return nil, err
	}
	// For group chats, fill in each sender's display name (cached per sender).
	if strings.HasSuffix(p.ChatJID, "@"+types.GroupServer) {
		names := map[string]string{}
		for i := range msgs {
			if msgs[i].FromMe || msgs[i].SenderJID == "" {
				continue
			}
			name, seen := names[msgs[i].SenderJID]
			if !seen {
				name = e.resolveContactName(msgs[i].SenderJID)
				names[msgs[i].SenderJID] = name
			}
			msgs[i].SenderName = name
		}
	}
	// Attach reactions to their messages.
	if reacts, err := e.db.ReactionsForChat(p.ChatJID); err == nil {
		for i := range msgs {
			if r := reacts[msgs[i].ID]; len(r) > 0 {
				msgs[i].Reactions = r
			}
		}
	}
	// Backfill pixel dimensions for cached images stored before dimensions were
	// recorded, so the GUI can reserve their layout space.
	for i := range msgs {
		if msgs[i].Type == "image" && msgs[i].MediaWidth == 0 && msgs[i].MediaPath != "" {
			if w, h := imageDimensions(msgs[i].MediaPath); w > 0 {
				msgs[i].MediaWidth = w
				msgs[i].MediaHeight = h
				_ = e.db.UpdateMediaDimensions(msgs[i].ChatJID, msgs[i].ID, w, h)
			}
		}
	}
	return msgs, nil
}

// imageDimensions reads the pixel size of a local image without decoding it fully.
func imageDimensions(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func (e *Engine) SendText(p ipc.SendTextParams) (interface{}, error) {
	jid, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	msg := buildTextMessage(p)
	resp, err := e.client.SendMessage(e.ctx, jid, msg)
	if err != nil {
		return nil, err
	}
	// Persist the outgoing message and broadcast it so every client (chat list
	// preview, ordering, and open conversation) updates consistently.
	sent := ipc.Message{
		ID:           resp.ID,
		ChatJID:      p.ChatJID,
		FromMe:       true,
		Timestamp:    resp.Timestamp.Unix(),
		Type:         "text",
		Text:         p.Text,
		Status:       "sent",
		QuotedID:     p.QuotedID,
		QuotedSender: p.QuotedSender,
		QuotedText:   p.QuotedText,
	}
	_ = e.db.PutMessage(sent)
	e.emit(ipc.NewEvent(ipc.EventMessage, sent))
	return map[string]any{"message_id": resp.ID, "timestamp": resp.Timestamp.Unix()}, nil
}

// buildTextMessage builds a plain text message, or a reply carrying the quoted
// message context when a quote target is set.
func buildTextMessage(p ipc.SendTextParams) *waE2E.Message {
	if p.QuotedID == "" {
		return &waE2E.Message{Conversation: proto.String(p.Text)}
	}
	ci := &waE2E.ContextInfo{
		StanzaID:      proto.String(p.QuotedID),
		QuotedMessage: &waE2E.Message{Conversation: proto.String(p.QuotedText)},
	}
	if p.QuotedSender != "" {
		ci.Participant = proto.String(p.QuotedSender)
	}
	return &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(p.Text),
			ContextInfo: ci,
		},
	}
}

// SetActiveChat records which chat is on screen so its incoming messages are
// not counted as unread, and clears any pending unread for it.
func (e *Engine) SetActiveChat(p ipc.SetActiveChatParams) (interface{}, error) {
	e.activeMu.Lock()
	e.activeChat = p.JID
	e.activeMu.Unlock()
	if p.JID != "" {
		e.clearUnread(p.JID)
	}
	return map[string]any{"ok": true}, nil
}

// bumpUnread increments the unread counter for an incoming message unless its
// chat is the one currently on screen, then notifies clients.
func (e *Engine) bumpUnread(msg ipc.Message) {
	if msg.FromMe {
		return
	}
	e.activeMu.Lock()
	active := e.activeChat
	e.activeMu.Unlock()
	if msg.ChatJID == active {
		return
	}
	count, err := e.db.IncrementUnread(msg.ChatJID)
	if err != nil {
		return
	}
	e.emit(ipc.NewEvent(ipc.EventChatUnread, map[string]any{
		"chat_jid": msg.ChatJID,
		"unread":   count,
	}))
}

// clearUnread zeroes a chat's unread counter and notifies clients.
func (e *Engine) clearUnread(chatJID string) {
	if chatJID == "" {
		return
	}
	if err := e.db.ResetUnread(chatJID); err != nil {
		return
	}
	e.emit(ipc.NewEvent(ipc.EventChatUnread, map[string]any{"chat_jid": chatJID, "unread": 0}))
}

// MarkRead sends read receipts for the given messages. Group chats need a per
// message participant and are not handled yet.
func (e *Engine) MarkRead(p ipc.MarkReadParams) (interface{}, error) {
	// Reading a chat clears its unread badge, groups included.
	e.clearUnread(p.ChatJID)
	if len(p.MessageIDs) == 0 {
		return map[string]any{"ok": true}, nil
	}
	chat, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	if chat.Server == types.GroupServer {
		return map[string]any{"skipped": "group"}, nil
	}
	ids := make([]types.MessageID, len(p.MessageIDs))
	for i, s := range p.MessageIDs {
		ids[i] = types.MessageID(s)
	}
	if err := e.client.MarkRead(e.ctx, ids, time.Now(), chat, chat); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// SetTyping sends a typing (composing) or paused chat presence for a chat.
func (e *Engine) SetTyping(p ipc.SetTypingParams) (interface{}, error) {
	jid, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	state := types.ChatPresencePaused
	if p.Composing {
		state = types.ChatPresenceComposing
	}
	if err := e.client.SendChatPresence(e.ctx, jid, state, types.ChatPresenceMediaText); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// SubscribePresence asks the server to push presence updates for a contact.
func (e *Engine) SubscribePresence(p ipc.SubscribePresenceParams) (interface{}, error) {
	jid, err := types.ParseJID(p.JID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	if err := e.client.SubscribePresence(e.ctx, jid); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// maybeDownloadMedia downloads images and stickers in the background and, when
// ready, records the cache path and notifies clients. Larger media (video,
// audio, documents) is left for on demand download.
func (e *Engine) maybeDownloadMedia(chatJID, id string, m *waE2E.Message) {
	if m == nil {
		return
	}
	if img := m.GetImageMessage(); img != nil {
		e.downloadMediaAsync(chatJID, id, img, sanitizeID(id)+extFromMime(img.GetMimetype(), ".jpg"))
	} else if st := m.GetStickerMessage(); st != nil {
		e.downloadMediaAsync(chatJID, id, st, sanitizeID(id)+".webp")
	}
}

func (e *Engine) downloadMediaAsync(chatJID, id string, media whatsmeow.DownloadableMessage, filename string) {
	go func() {
		data, err := e.client.Download(e.ctx, media)
		if err != nil {
			log.Printf("download media %s: %v", id, err)
			return
		}
		dir, err := store.MediaDir()
		if err != nil {
			log.Printf("media dir: %v", err)
			return
		}
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			log.Printf("write media %s: %v", id, err)
			return
		}
		_ = e.db.UpdateMediaPath(chatJID, id, path)
		e.emit(ipc.NewEvent(ipc.EventMessageMedia, map[string]any{
			"chat_jid":   chatJID,
			"id":         id,
			"media_path": path,
		}))
	}()
}

// DownloadMedia fetches a message's attachment on demand from the stored
// protobuf, caches it, and notifies clients with the local path so the GUI can
// open it. Only messages received after media info was stored can be fetched.
func (e *Engine) DownloadMedia(p ipc.DownloadMediaParams) (interface{}, error) {
	raw, typ, err := e.db.MediaInfo(p.ChatJID, p.MessageID)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("no downloadable media stored for %s", p.MessageID)
	}
	var media whatsmeow.DownloadableMessage
	var filename string
	switch typ {
	case "image":
		m := &waE2E.ImageMessage{}
		if err := proto.Unmarshal(raw, m); err != nil {
			return nil, err
		}
		media, filename = m, sanitizeID(p.MessageID)+extFromMime(m.GetMimetype(), ".jpg")
	case "sticker":
		m := &waE2E.StickerMessage{}
		if err := proto.Unmarshal(raw, m); err != nil {
			return nil, err
		}
		media, filename = m, sanitizeID(p.MessageID)+".webp"
	case "video":
		m := &waE2E.VideoMessage{}
		if err := proto.Unmarshal(raw, m); err != nil {
			return nil, err
		}
		media, filename = m, sanitizeID(p.MessageID)+extFromMime(m.GetMimetype(), ".mp4")
	case "audio":
		m := &waE2E.AudioMessage{}
		if err := proto.Unmarshal(raw, m); err != nil {
			return nil, err
		}
		media, filename = m, sanitizeID(p.MessageID)+extFromMime(m.GetMimetype(), ".ogg")
	case "document":
		m := &waE2E.DocumentMessage{}
		if err := proto.Unmarshal(raw, m); err != nil {
			return nil, err
		}
		media, filename = m, documentFilename(p.MessageID, m)
	default:
		return nil, fmt.Errorf("cannot download media of type %q", typ)
	}
	e.downloadMediaAsync(p.ChatJID, p.MessageID, media, filename)
	return map[string]any{"ok": true}, nil
}

// documentFilename builds a safe cache filename that keeps the document's
// extension so the system handler opens it with the right application.
func documentFilename(id string, m *waE2E.DocumentMessage) string {
	ext := filepath.Ext(m.GetFileName())
	if ext == "" {
		ext = extFromMime(m.GetMimetype(), ".bin")
	}
	return sanitizeID(id) + ext
}

func extFromMime(mime, def string) string {
	switch {
	case strings.Contains(mime, "png"):
		return ".png"
	case strings.Contains(mime, "webp"):
		return ".webp"
	case strings.Contains(mime, "gif"):
		return ".gif"
	case strings.Contains(mime, "jpeg"), strings.Contains(mime, "jpg"):
		return ".jpg"
	case strings.Contains(mime, "mp4"):
		return ".mp4"
	case strings.Contains(mime, "webm"):
		return ".webm"
	case strings.Contains(mime, "3gpp"):
		return ".3gp"
	case strings.Contains(mime, "quicktime"):
		return ".mov"
	case strings.Contains(mime, "ogg"):
		return ".ogg"
	case strings.Contains(mime, "mpeg"), strings.Contains(mime, "mp3"):
		return ".mp3"
	case strings.Contains(mime, "wav"):
		return ".wav"
	case strings.Contains(mime, "aac"):
		return ".aac"
	case strings.Contains(mime, "pdf"):
		return ".pdf"
	}
	return def
}

func sanitizeID(id string) string {
	return strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(id)
}

// DeleteMessage revokes one of our own messages for everyone.
func (e *Engine) DeleteMessage(p ipc.DeleteMessageParams) (interface{}, error) {
	chat, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	revoke := e.client.BuildRevoke(chat, types.EmptyJID, types.MessageID(p.MessageID))
	if _, err := e.client.SendMessage(e.ctx, chat, revoke); err != nil {
		return nil, err
	}
	_ = e.db.MarkRevoked(p.ChatJID, p.MessageID)
	e.emit(ipc.NewEvent(ipc.EventMessageRevoked, map[string]any{
		"chat_jid":   p.ChatJID,
		"message_id": p.MessageID,
	}))
	return map[string]any{"ok": true}, nil
}

// EditMessage replaces the text of one of our own messages and notifies clients.
func (e *Engine) EditMessage(p ipc.EditMessageParams) (interface{}, error) {
	chat, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	newContent := &waE2E.Message{Conversation: proto.String(p.Text)}
	edit := e.client.BuildEdit(chat, types.MessageID(p.MessageID), newContent)
	if _, err := e.client.SendMessage(e.ctx, chat, edit); err != nil {
		return nil, err
	}
	_ = e.db.UpdateText(p.ChatJID, p.MessageID, p.Text)
	e.emit(ipc.NewEvent(ipc.EventMessageEdited, map[string]any{
		"chat_jid":   p.ChatJID,
		"message_id": p.MessageID,
		"text":       p.Text,
	}))
	return map[string]any{"ok": true}, nil
}

// SendReaction reacts to a message with an emoji, or removes the reaction when
// the emoji is empty.
func (e *Engine) SendReaction(p ipc.SendReactionParams) (interface{}, error) {
	chat, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	var sender types.JID
	switch {
	case p.FromMe && e.client.Store.ID != nil:
		sender = *e.client.Store.ID
	case p.SenderJID != "":
		if sender, err = types.ParseJID(p.SenderJID); err != nil {
			return nil, fmt.Errorf("invalid sender: %w", err)
		}
	default:
		sender = chat
	}
	reaction := e.client.BuildReaction(chat, sender, types.MessageID(p.MessageID), p.Emoji)
	if _, err := e.client.SendMessage(e.ctx, chat, reaction); err != nil {
		return nil, err
	}
	ownID := ""
	if e.client.Store.ID != nil {
		ownID = e.client.Store.ID.ToNonAD().String()
	}
	_ = e.db.PutReaction(p.ChatJID, p.MessageID, ownID, p.Emoji)
	e.emit(ipc.NewEvent(ipc.EventReaction, map[string]any{
		"chat_jid":   p.ChatJID,
		"message_id": p.MessageID,
		"sender_jid": ownID,
		"emoji":      p.Emoji,
		"from_me":    true,
	}))
	return map[string]any{"ok": true}, nil
}

// SendMedia uploads a local file and sends it as an image or a document.
func (e *Engine) SendMedia(p ipc.SendMediaParams) (interface{}, error) {
	jid, err := types.ParseJID(p.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("invalid jid: %w", err)
	}
	data, err := os.ReadFile(p.FilePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	mimeType := http.DetectContentType(data)

	var msg *waE2E.Message
	msgType := "document"
	if strings.HasPrefix(mimeType, "image/") {
		up, upErr := e.client.Upload(e.ctx, data, whatsmeow.MediaImage)
		if upErr != nil {
			return nil, fmt.Errorf("upload: %w", upErr)
		}
		img := &waE2E.ImageMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
		}
		if p.Caption != "" {
			img.Caption = proto.String(p.Caption)
		}
		msg = &waE2E.Message{ImageMessage: img}
		msgType = "image"
	} else {
		up, upErr := e.client.Upload(e.ctx, data, whatsmeow.MediaDocument)
		if upErr != nil {
			return nil, fmt.Errorf("upload: %w", upErr)
		}
		doc := &waE2E.DocumentMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			FileName:      proto.String(filepath.Base(p.FilePath)),
		}
		if p.Caption != "" {
			doc.Caption = proto.String(p.Caption)
		}
		msg = &waE2E.Message{DocumentMessage: doc}
	}

	resp, err := e.client.SendMessage(e.ctx, jid, msg)
	if err != nil {
		return nil, err
	}
	sent := ipc.Message{
		ID:        resp.ID,
		ChatJID:   p.ChatJID,
		FromMe:    true,
		Timestamp: resp.Timestamp.Unix(),
		Type:      msgType,
		Text:      p.Caption,
		Status:    "sent",
	}
	if msgType == "image" {
		sent.MediaPath = p.FilePath // show the local file inline right away
	} else if p.Caption == "" {
		sent.Text = filepath.Base(p.FilePath)
	}
	_ = e.db.PutMessage(sent)
	e.emit(ipc.NewEvent(ipc.EventMessage, sent))
	return map[string]any{"message_id": resp.ID}, nil
}
