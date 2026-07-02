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
    unread       INTEGER NOT NULL DEFAULT 0
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

const insertMessageSQL = `INSERT OR IGNORE INTO messages
    (id, chat_jid, sender_jid, from_me, ts, type, text, status, media_path) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

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
		if _, err := insMsg.Exec(m.ID, m.ChatJID, m.SenderJID, boolToInt(m.FromMe), m.Timestamp, m.Type, m.Text, m.Status, m.MediaPath); err != nil {
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

// UpdateMediaPath records the local cache path of a downloaded media message.
func (d *DB) UpdateMediaPath(chatJID, id, path string) error {
	_, err := d.sql.Exec(`UPDATE messages SET media_path = ? WHERE chat_jid = ? AND id = ?`, path, chatJID, id)
	return err
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
		`SELECT jid, name, is_group, last_ts, last_preview, unread
		 FROM chats ORDER BY last_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	chats := []ipc.Chat{}
	for rows.Next() {
		var c ipc.Chat
		var isGroup, unread int
		if err := rows.Scan(&c.JID, &c.Name, &isGroup, &c.LastTS, &c.LastPreview, &unread); err != nil {
			return nil, err
		}
		c.IsGroup = isGroup != 0
		c.UnreadCount = unread
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
		`SELECT id, chat_jid, sender_jid, from_me, ts, type, text, status, media_path
		 FROM messages WHERE chat_jid = ? ORDER BY ts DESC LIMIT ?`,
		chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	msgs := []ipc.Message{}
	for rows.Next() {
		var m ipc.Message
		var fromMe int
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.SenderJID, &fromMe, &m.Timestamp, &m.Type, &m.Text, &m.Status, &m.MediaPath); err != nil {
			return nil, err
		}
		m.FromMe = fromMe != 0
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
