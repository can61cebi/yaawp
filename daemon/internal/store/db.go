package store

import (
	"database/sql"
	"fmt"
	"strings"

	"cebi.tr/yaawp/internal/ipc"

	_ "github.com/mattn/go-sqlite3"
)

// DB is the application data store for chats and messages. It is separate from
// the whatsmeow session store so the two schemas evolve independently.
type DB struct {
	sql *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS chats (
    jid          TEXT PRIMARY KEY,
    name         TEXT NOT NULL DEFAULT '',
    is_group     INTEGER NOT NULL DEFAULT 0,
    last_ts      INTEGER NOT NULL DEFAULT 0,
    last_preview TEXT NOT NULL DEFAULT '',
    unread       INTEGER NOT NULL DEFAULT 0,
    pinned       INTEGER NOT NULL DEFAULT 0,
    muted        INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS messages (
    id         TEXT NOT NULL,
    chat_jid   TEXT NOT NULL,
    sender_jid TEXT NOT NULL DEFAULT '',
    from_me    INTEGER NOT NULL DEFAULT 0,
    ts         INTEGER NOT NULL DEFAULT 0,
    type       TEXT NOT NULL DEFAULT 'text',
    text       TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT '',
    media_path TEXT NOT NULL DEFAULT '',
    media_w    INTEGER NOT NULL DEFAULT 0,
    media_h    INTEGER NOT NULL DEFAULT 0,
    raw_media  BLOB,
    edited     INTEGER NOT NULL DEFAULT 0,
    starred    INTEGER NOT NULL DEFAULT 0,
    quoted_id     TEXT NOT NULL DEFAULT '',
    quoted_sender TEXT NOT NULL DEFAULT '',
    quoted_text   TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (chat_jid, id)
);
CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON messages (chat_jid, ts);
CREATE TABLE IF NOT EXISTS reactions (
    chat_jid   TEXT NOT NULL,
    message_id TEXT NOT NULL,
    sender_jid TEXT NOT NULL,
    emoji      TEXT NOT NULL,
    PRIMARY KEY (chat_jid, message_id, sender_jid)
);
`

const insertMessageSQL = `INSERT INTO messages
    (id, chat_jid, sender_jid, from_me, ts, type, text, status, media_path, media_w, media_h, raw_media, edited, quoted_id, quoted_sender, quoted_text)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(chat_jid, id) DO UPDATE SET
        raw_media = COALESCE(messages.raw_media, excluded.raw_media),
        media_w = CASE WHEN messages.media_w = 0 THEN excluded.media_w ELSE messages.media_w END,
        media_h = CASE WHEN messages.media_h = 0 THEN excluded.media_h ELSE messages.media_h END`

// updateChatSQL advances a chat summary only when the incoming message is at
// least as recent as the stored one.
const updateChatSQL = `INSERT INTO chats (jid, last_ts, last_preview) VALUES (?, ?, ?)
    ON CONFLICT(jid) DO UPDATE SET
      last_ts = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_ts ELSE chats.last_ts END,
      last_preview = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_preview ELSE chats.last_preview END`

// OpenDB opens the application database under the XDG data directory, applies
// the schema and runs migrations.
func OpenDB() (*DB, error) {
	dir, err := DataDir()
	if err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s/yaawp.db?_foreign_keys=on", dir)
	sqldb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := sqldb.Exec(schema); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	db := &DB{sql: sqldb}
	db.migrate()
	return db, nil
}

// migrate adds columns that may be missing from an older database. Errors are
// ignored because the column usually already exists.
func (d *DB) migrate() {
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN status TEXT NOT NULL DEFAULT ''`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN media_path TEXT NOT NULL DEFAULT ''`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN quoted_id TEXT NOT NULL DEFAULT ''`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN quoted_sender TEXT NOT NULL DEFAULT ''`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN quoted_text TEXT NOT NULL DEFAULT ''`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN media_w INTEGER NOT NULL DEFAULT 0`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN media_h INTEGER NOT NULL DEFAULT 0`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN raw_media BLOB`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN edited INTEGER NOT NULL DEFAULT 0`)
	_, _ = d.sql.Exec(`ALTER TABLE chats ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`)
	_, _ = d.sql.Exec(`ALTER TABLE chats ADD COLUMN muted INTEGER NOT NULL DEFAULT 0`)
	_, _ = d.sql.Exec(`ALTER TABLE messages ADD COLUMN starred INTEGER NOT NULL DEFAULT 0`)
}

// Close releases the database handle.
func (d *DB) Close() error { return d.sql.Close() }

// PutMessage inserts a single message and refreshes its chat summary.
func (d *DB) PutMessage(m ipc.Message) error {
	return d.PutMessages([]ipc.Message{m})
}

// PutMessages inserts many messages and refreshes chat summaries in one
// transaction. This keeps bulk history sync fast.
func (d *DB) PutMessages(msgs []ipc.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	insMsg, err := tx.Prepare(insertMessageSQL)
	if err != nil {
		return err
	}
	defer func() { _ = insMsg.Close() }()

	upChat, err := tx.Prepare(updateChatSQL)
	if err != nil {
		return err
	}
	defer func() { _ = upChat.Close() }()

	for _, m := range msgs {
		if _, err := insMsg.Exec(m.ID, m.ChatJID, m.SenderJID, boolToInt(m.FromMe), m.Timestamp, m.Type, m.Text, m.Status, m.MediaPath, m.MediaWidth, m.MediaHeight, m.RawMedia, boolToInt(m.Edited), m.QuotedID, m.QuotedSender, m.QuotedText); err != nil {
			return err
		}
		if _, err := upChat.Exec(m.ChatJID, m.Timestamp, m.Text); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateStatus advances the delivery status of the given outgoing messages. A
// read status is never downgraded back to delivered.
func (d *DB) UpdateStatus(chatJID string, ids []string, status string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	query := `UPDATE messages SET status = ? WHERE chat_jid = ? AND id IN (` + placeholders + `) AND status != 'read'`
	args := make([]any, 0, len(ids)+2)
	args = append(args, status, chatJID)
	for _, id := range ids {
		args = append(args, id)
	}
	_, err := d.sql.Exec(query, args...)
	return err
}

// PutReaction stores or removes a reaction. An empty emoji removes it.
func (d *DB) PutReaction(chatJID, messageID, senderJID, emoji string) error {
	if emoji == "" {
		_, err := d.sql.Exec(
			`DELETE FROM reactions WHERE chat_jid = ? AND message_id = ? AND sender_jid = ?`,
			chatJID, messageID, senderJID)
		return err
	}
	_, err := d.sql.Exec(
		`INSERT INTO reactions (chat_jid, message_id, sender_jid, emoji) VALUES (?, ?, ?, ?)
		 ON CONFLICT(chat_jid, message_id, sender_jid) DO UPDATE SET emoji = excluded.emoji`,
		chatJID, messageID, senderJID, emoji)
	return err
}

// ReactionsForChat returns a map of message id to sender jid to emoji.
func (d *DB) ReactionsForChat(chatJID string) (map[string]map[string]string, error) {
	rows, err := d.sql.Query(
		`SELECT message_id, sender_jid, emoji FROM reactions WHERE chat_jid = ?`, chatJID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string]map[string]string{}
	for rows.Next() {
		var mid, sender, emoji string
		if err := rows.Scan(&mid, &sender, &emoji); err != nil {
			return nil, err
		}
		if out[mid] == nil {
			out[mid] = map[string]string{}
		}
		out[mid][sender] = emoji
	}
	return out, rows.Err()
}

// MarkRevoked marks a message as deleted for everyone.
func (d *DB) MarkRevoked(chatJID, id string) error {
	_, err := d.sql.Exec(
		`UPDATE messages SET type = 'revoked', text = '', media_path = '' WHERE chat_jid = ? AND id = ?`,
		chatJID, id)
	return err
}

// UpdateText replaces a message's text and marks it as edited.
func (d *DB) UpdateText(chatJID, id, text string) error {
	_, err := d.sql.Exec(`UPDATE messages SET text = ?, edited = 1 WHERE chat_jid = ? AND id = ?`, text, chatJID, id)
	return err
}

// UpdateMediaPath records the local cache path of a downloaded media message.
func (d *DB) UpdateMediaPath(chatJID, id, path string) error {
	_, err := d.sql.Exec(`UPDATE messages SET media_path = ? WHERE chat_jid = ? AND id = ?`, path, chatJID, id)
	return err
}

// UpdateMediaDimensions records the pixel size of a media message.
func (d *DB) UpdateMediaDimensions(chatJID, id string, w, h int) error {
	_, err := d.sql.Exec(`UPDATE messages SET media_w = ?, media_h = ? WHERE chat_jid = ? AND id = ?`, w, h, chatJID, id)
	return err
}

// MediaInfo returns the message type and the stored protobuf submessage needed
// to download the attachment on demand. raw is nil when nothing was stored.
func (d *DB) MediaInfo(chatJID, id string) (raw []byte, typ string, err error) {
	row := d.sql.QueryRow(`SELECT type, raw_media FROM messages WHERE chat_jid = ? AND id = ?`, chatJID, id)
	err = row.Scan(&typ, &raw)
	return raw, typ, err
}

// IncrementUnread bumps a chat's unread counter and returns the new value.
func (d *DB) IncrementUnread(jid string) (int, error) {
	if _, err := d.sql.Exec(`UPDATE chats SET unread = unread + 1 WHERE jid = ?`, jid); err != nil {
		return 0, err
	}
	var n int
	err := d.sql.QueryRow(`SELECT unread FROM chats WHERE jid = ?`, jid).Scan(&n)
	return n, err
}

// ResetUnread clears a chat's unread counter.
func (d *DB) ResetUnread(jid string) error {
	_, err := d.sql.Exec(`UPDATE chats SET unread = 0 WHERE jid = ?`, jid)
	return err
}

// SetPinned pins or unpins a chat so it sorts to the top of the list.
func (d *DB) SetPinned(jid string, pinned bool) error {
	_, err := d.sql.Exec(`UPDATE chats SET pinned = ? WHERE jid = ?`, boolToInt(pinned), jid)
	return err
}

// SetMuted mutes or unmutes a chat's notifications.
func (d *DB) SetMuted(jid string, muted bool) error {
	_, err := d.sql.Exec(`UPDATE chats SET muted = ? WHERE jid = ?`, boolToInt(muted), jid)
	return err
}

// SetStarred stars or unstars a message.
func (d *DB) SetStarred(chatJID, id string, starred bool) error {
	_, err := d.sql.Exec(`UPDATE messages SET starred = ? WHERE chat_jid = ? AND id = ?`, boolToInt(starred), chatJID, id)
	return err
}

// ListStarred returns starred messages across all chats, newest first, with the
// chat name attached for display.
func (d *DB) ListStarred() ([]ipc.Message, error) {
	rows, err := d.sql.Query(
		`SELECT m.id, m.chat_jid, m.sender_jid, m.from_me, m.ts, m.type, m.text, m.media_path, COALESCE(c.name, '')
		 FROM messages m LEFT JOIN chats c ON c.jid = m.chat_jid
		 WHERE m.starred = 1 ORDER BY m.ts DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	msgs := []ipc.Message{}
	for rows.Next() {
		var m ipc.Message
		var fromMe int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.SenderJID, &fromMe, &m.Timestamp, &m.Type, &m.Text, &m.MediaPath, &m.ChatName); err != nil {
			return nil, err
		}
		m.FromMe = fromMe != 0
		m.Starred = true
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// SetChatName sets the display name and group flag for a chat.
func (d *DB) SetChatName(jid, name string, isGroup bool) error {
	_, err := d.sql.Exec(
		`INSERT INTO chats (jid, name, is_group) VALUES (?, ?, ?)
		 ON CONFLICT(jid) DO UPDATE SET name = excluded.name, is_group = excluded.is_group`,
		jid, name, boolToInt(isGroup),
	)
	return err
}

// ListChats returns all known chats, most recent first.
func (d *DB) ListChats() ([]ipc.Chat, error) {
	rows, err := d.sql.Query(
		`SELECT jid, name, is_group, last_ts, last_preview, unread, pinned, muted
		 FROM chats ORDER BY pinned DESC, last_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	chats := []ipc.Chat{}
	for rows.Next() {
		var c ipc.Chat
		var isGroup, unread, pinned, muted int
		if err := rows.Scan(&c.JID, &c.Name, &isGroup, &c.LastTS, &c.LastPreview, &unread, &pinned, &muted); err != nil {
			return nil, err
		}
		c.IsGroup = isGroup != 0
		c.UnreadCount = unread
		c.Pinned = pinned != 0
		c.Muted = muted != 0
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

// ListMessages returns up to limit messages for a chat in chronological order.
func (d *DB) ListMessages(chatJID string, limit int) ([]ipc.Message, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := d.sql.Query(
		`SELECT id, chat_jid, sender_jid, from_me, ts, type, text, status, media_path, media_w, media_h, edited, starred, quoted_id, quoted_sender, quoted_text
		 FROM messages WHERE chat_jid = ? ORDER BY ts DESC LIMIT ?`,
		chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	msgs := []ipc.Message{}
	for rows.Next() {
		var m ipc.Message
		var fromMe, edited, starred int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.SenderJID, &fromMe, &m.Timestamp, &m.Type, &m.Text, &m.Status, &m.MediaPath, &m.MediaWidth, &m.MediaHeight, &edited, &starred, &m.QuotedID, &m.QuotedSender, &m.QuotedText); err != nil {
			return nil, err
		}
		m.FromMe = fromMe != 0
		m.Edited = edited != 0
		m.Starred = starred != 0
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// The query is newest first; reverse to ascending chronological order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
