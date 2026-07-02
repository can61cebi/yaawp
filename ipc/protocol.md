# yaawp IPC protocol

The single contract between the daemon (Go) and the GUI (Kirigami/C++). Both
sides conform to this document.

- Transport: Unix domain socket at `$XDG_RUNTIME_DIR/yaawp/daemon.sock`
- Framing: one JSON object per line, terminated by `\n` (newline-delimited JSON)
- Directions:
  - GUI to daemon: Command (request), correlated by `id`
  - daemon to GUI: Response (reply to a Command) and Event (unsolicited push)

## Envelope

```json
{ "type": "cmd" | "resp" | "event", ... }
```

## Command (GUI to daemon)

```json
{"type":"cmd","id":"7","method":"send_text","params":{"chat_jid":"90555...@s.whatsapp.net","text":"hello"}}
```

| method          | params                          | description                    |
|-----------------|---------------------------------|--------------------------------|
| get_state       | none                            | session and connection state   |
| login           | none                            | begins QR pairing              |
| logout          | none                            | unlinks this device            |
| list_chats      | none                            | chat list (currently empty)    |
| list_messages   | {chat_jid, limit, before?}      | chat history (currently empty) |
| send_text       | {chat_jid, text}                | sends a text message           |
| mark_read       | {chat_jid, message_ids[]}       | read receipt                   |

## Response (daemon to GUI)

```json
{"type":"resp","id":"7","ok":true,"result":{"message_id":"3EB0...","timestamp":1720000000}}
{"type":"resp","id":"5","ok":false,"error":"not_logged_in"}
```

## Event (daemon to GUI, unsolicited)

```json
{"type":"event","event":"qr","data":{"code":"2@abc..."}}
```

| event         | data                                                              |
|---------------|------------------------------------------------------------------|
| qr            | {code}: raw string to render as a QR image                       |
| pair_success  | {jid, push_name}                                                 |
| connection    | {state}: connecting, connected, disconnected, logged_out         |
| message       | {id, chat_jid, sender_jid, from_me, timestamp, type, text}       |
| receipt       | {chat_jid, message_ids[], receipt}: delivered, read              |
| presence      | {jid, state}: available, unavailable                             |
| history_sync  | {progress}: 0 to 100                                             |

## Data models

Chat: `{jid, name, is_group, last_message_ts, last_message_preview, unread_count}`

Message: `{id, chat_jid, sender_jid, from_me, timestamp, type, text}`
