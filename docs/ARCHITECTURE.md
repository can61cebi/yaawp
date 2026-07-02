# Architecture

yaawp is split into two processes that communicate over a local Unix socket.

```
+-------------------------------------------+
|  Kirigami / QML + C++   (GUI process)     |   native KDE application
|  chat list, conversation, notifications   |
+---------------------+---------------------+
                      | IPC (Unix socket, newline-delimited JSON)
+---------------------+---------------------+
|  Go daemon   (whatsmeow engine)           |   headless service
|  Noise, libsignal, protobuf, SQLite       |
+---------------------+---------------------+
                      | WebSocket (wss)
              +-------+--------+
              | WhatsApp multi |
              | device servers |
              +----------------+
```

## Why two processes

The protocol engine is written in Go because `whatsmeow` is the most mature
implementation of the WhatsApp multi-device protocol. The GUI is written in
C++ with Kirigami and QML so the application is native to KDE Plasma.

Binding Go into the C++ process through cgo is fragile: the Go scheduler, the
Qt event loop, callback marshalling and garbage collection interact badly.
Two processes keep the language boundary clean. The daemon can run as a
systemd user service and hold the session even if the GUI restarts.

## Components

- `daemon/` Go service.
  - `internal/engine` wraps the whatsmeow client and implements `ipc.Backend`.
  - `internal/ipc` defines the wire protocol and the Unix socket server.
  - `internal/store` resolves XDG data and socket paths.
  - `cmd/yaawp-daemon` wires everything together.
- `gui/` C++ and QML application.
  - `IpcClient` owns the socket and exposes signals and invokable methods.
  - `Controller` exposes login and connection state to QML.
  - `ChatListModel` and `MessageModel` are the list models.
  - `qml/` holds the Kirigami pages.
- `ipc/protocol.md` is the single source of truth for the wire format.

## IPC

See `ipc/protocol.md`. The transport is one JSON object per line over a Unix
socket at `$XDG_RUNTIME_DIR/yaawp/daemon.sock`. Commands are correlated by id.
Events are pushed to every connected client.

## Data and secrets

The whatsmeow session store is an SQLite database under
`$XDG_DATA_HOME/yaawp/session.db`. A later step will move session secrets into
KWallet and cache media under `$XDG_CACHE_HOME/yaawp`.
