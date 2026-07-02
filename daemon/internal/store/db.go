package store

import (
	"database/sql"
	"fmt"

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
    PRIMARY KEY (chat_jid, id)
);
CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON messages (chat_jid, ts);
`

// OpenDB opens the application database under the XDG data directory and
// applies the schema.
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
	return &DB{sql: sqldb}, nil
}

// Close releases the database handle.
func (d *DB) Close() error { return d.sql.Close() }

// PutMessage inserts a message and refreshes its chat summary row.
func (d *DB) PutMessage(m ipc.Message) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		`INSERT OR IGNORE INTO messages (id, chat_jid, sender_jid, from_me, ts, type, text)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ChatJID, m.SenderJID, boolToInt(m.FromMe), m.Timestamp, m.Type, m.Text,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO chats (jid, last_ts, last_preview) VALUES (?, ?, ?)
		 ON CONFLICT(jid) DO UPDATE SET
		   last_ts = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_ts ELSE chats.last_ts END,
		   last_preview = CASE WHEN excluded.last_ts >= chats.last_ts THEN excluded.last_preview ELSE chats.last_preview END`,
		m.ChatJID, m.Timestamp, m.Text,
	); err != nil {
		return err
	}
	return tx.Commit()
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
		`SELECT id, chat_jid, sender_jid, from_me, ts, type, text
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
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.SenderJID, &fromMe, &m.Timestamp, &m.Type, &m.Text); err != nil {
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
