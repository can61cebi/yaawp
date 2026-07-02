package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"sync"
)

var errUnknownMethod = errors.New("unknown_method")

// client represents one connected GUI. Writes are serialised with a mutex.
type client struct {
	mu  sync.Mutex
	enc *json.Encoder
}

func (c *client) write(v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// json.Encoder appends a trailing newline, giving the newline-delimited framing.
	_ = c.enc.Encode(v)
}

// Server is a newline-delimited JSON IPC server over a Unix socket.
type Server struct {
	path    string
	backend Backend

	mu      sync.RWMutex
	clients map[*client]struct{}
}

// NewServer creates a server bound to the given socket path and backend.
func NewServer(path string, backend Backend) *Server {
	return &Server{path: path, backend: backend, clients: map[*client]struct{}{}}
}

// Broadcast pushes an event to every connected client. The engine uses this as
// its event sink.
func (s *Server) Broadcast(evt Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for c := range s.clients {
		c.write(evt)
	}
}

// Serve listens on the socket and blocks until ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	_ = os.Remove(s.path) // clear any stale socket
	ln, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	log.Printf("IPC listening on %s", s.path)

	go func() {
		<-ctx.Done()
		_ = ln.Close()
		_ = os.Remove(s.path)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	c := &client{enc: json.NewEncoder(conn)}
	s.mu.Lock()
	s.clients[c] = struct{}{}
	n := len(s.clients)
	s.mu.Unlock()
	log.Printf("client connected (%d total)", n)

	// Send a snapshot so a client that connects late still sees the current
	// state and any pending QR code.
	for _, evt := range s.backend.InitialEvents() {
		c.write(evt)
	}

	defer func() {
		s.mu.Lock()
		delete(s.clients, c)
		s.mu.Unlock()
		_ = conn.Close()
	}()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024) // allow large lines
	for scanner.Scan() {
		var cmd Command
		if err := json.Unmarshal(scanner.Bytes(), &cmd); err != nil {
			c.write(Response{Type: TypeResponse, OK: false, Error: "bad_json"})
			continue
		}
		s.dispatch(c, cmd)
	}
}

func (s *Server) dispatch(c *client, cmd Command) {
	resp := Response{Type: TypeResponse, ID: cmd.ID, OK: true}
	var (
		result interface{}
		err    error
	)
	switch cmd.Method {
	case MethodGetState:
		result, err = s.backend.GetState()
	case MethodLogin:
		result, err = s.backend.Login()
	case MethodLogout:
		result, err = s.backend.Logout()
	case MethodListChats:
		result, err = s.backend.ListChats()
	case MethodListMessages:
		var p ListMessagesParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.ListMessages(p)
	case MethodSendText:
		var p SendTextParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SendText(p)
	case MethodMarkRead:
		var p MarkReadParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.MarkRead(p)
	case MethodSetTyping:
		var p SetTypingParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SetTyping(p)
	case MethodSubscribePresence:
		var p SubscribePresenceParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SubscribePresence(p)
	case MethodDeleteMessage:
		var p DeleteMessageParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.DeleteMessage(p)
	case MethodSendReaction:
		var p SendReactionParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SendReaction(p)
	case MethodSendMedia:
		var p SendMediaParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SendMedia(p)
	case MethodDownloadMedia:
		var p DownloadMediaParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.DownloadMedia(p)
	case MethodSetActiveChat:
		var p SetActiveChatParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SetActiveChat(p)
	case MethodEditMessage:
		var p EditMessageParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.EditMessage(p)
	case MethodGroupInfo:
		var p GroupInfoParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.GroupInfo(p)
	case MethodContactInfo:
		var p ContactInfoParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.ContactInfo(p)
	case MethodSetDisappearing:
		var p SetDisappearingParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SetDisappearing(p)
	case MethodSetBlocked:
		var p SetBlockedParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SetBlocked(p)
	case MethodSetPinned:
		var p SetPinnedParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SetPinned(p)
	case MethodSetMuted:
		var p SetMutedParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.SetMuted(p)
	case MethodStarMessage:
		var p StarMessageParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.StarMessage(p)
	case MethodListStarred:
		result, err = s.backend.ListStarred()
	case MethodRequestAvatar:
		var p RequestAvatarParams
		_ = json.Unmarshal(cmd.Params, &p)
		result, err = s.backend.RequestAvatar(p)
	default:
		err = errUnknownMethod
	}
	if err != nil {
		resp.OK = false
		resp.Error = err.Error()
	} else {
		resp.Result = result
	}
	c.write(resp)
}
