# yaawp

A native WhatsApp client for Arch Linux and KDE Plasma. yaawp speaks the
WhatsApp multi-device protocol directly through a background service and
presents it in a native Kirigami interface. It is not a web wrapper and it does
not embed a browser.

## Features

- Link to your account by scanning a QR code, the same way the official desktop
  and web clients pair a linked device.
- Send and receive text, replies, and reactions.
- Send and receive images, video, documents, and voice messages, with
  downloads saved to a local cache.
- Record voice messages from the app.
- Edit and delete your own messages, and star messages to find them later.
- Pin and mute chats.
- View group and contact information, and set disappearing messages.
- Block and unblock contacts.
- See presence: online, typing, and last seen.
- Link previews, in-chat search with a match count and highlight, and
  adjustable message text size.
- Desktop notifications for incoming messages, an unread badge, and a system
  tray icon. Closing the window keeps yaawp running in the tray.
- Start on login and open straight into the tray so notifications arrive in the
  background.
- Privacy controls for read receipts and online and typing status.

## Design

yaawp runs as two processes:

1. A headless Go daemon built on the whatsmeow library. It holds the session,
   performs the Noise handshake and Signal end to end encryption, and stores
   state in SQLite.
2. A native C++ and QML application built with Qt 6 and Kirigami. It talks to
   the daemon over a local Unix socket using a small newline-delimited JSON
   protocol.

The two processes keep the Go engine and the Qt event loop cleanly separated.
The GUI launches the daemon on demand and reconnects if the link drops, and the
daemon refuses to run twice so a single session stays authoritative. See
docs/ARCHITECTURE.md and ipc/protocol.md for details.

## Requirements

Runtime:

- qt6-base, qt6-declarative, qt6-multimedia
- kirigami, prison, qqc2-desktop-style
- kstatusnotifieritem, knotifications

Build:

- go
- cmake, extra-cmake-modules, ninja
- a C and C++ toolchain

On Arch Linux:

```
sudo pacman -S --needed go cmake extra-cmake-modules ninja \
    qt6-base qt6-declarative qt6-multimedia \
    kirigami prison qqc2-desktop-style kstatusnotifieritem knotifications
```

## Install

### Local install with the script

The install script builds the daemon and the GUI and installs them into a user
prefix (default `~/.local`). No root is needed.

```
git clone https://github.com/can61cebi/yaawp.git
cd yaawp
./install.sh
```

After it finishes, yaawp appears in the application launcher (search for
"yaawp"). If `~/.local/bin` is not on your `PATH`, add it so the `yaawp` command
resolves:

```
export PATH="$HOME/.local/bin:$PATH"
```

To remove it again:

```
./install.sh --uninstall
```

Your session data in `~/.local/share/yaawp` is left in place so a reinstall
keeps you logged in. Delete it by hand for a clean slate.

### Arch package

A PKGBUILD is in `packaging/`. It builds from a tagged release and installs
system wide, along with an optional systemd user service for the daemon.

```
cd packaging
makepkg -si
```

### Manual build

Daemon:

```
cd daemon
make tidy
make build
```

GUI:

```
cd gui
cmake -B build -G Ninja -DCMAKE_BUILD_TYPE=Release
cmake --build build
```

Run the GUI directly with `./gui/build/bin/yaawp`. It launches the daemon from
`PATH` or, failing that, from the build tree next to it.

## First run

Start yaawp from the launcher or the terminal. On first run the daemon is not
paired, so the window shows a QR code. Open WhatsApp on your phone, go to
Settings, Linked Devices, Link a Device, and scan the code. Once linked, your
chats sync in and the client is ready.

## Start on login and notifications

Open Settings and turn on "Start on login". yaawp then writes an autostart entry
and, on your next login, opens directly into the system tray. The daemon starts
with it, so incoming messages raise desktop notifications and update the tray
badge even while the window is closed. Click the tray icon to open the window.

If you prefer to run the daemon as a system service instead, enable the unit
shipped with the Arch package:

```
systemctl --user enable --now yaawp-daemon.service
```

The daemon only runs once regardless of how it is started, so the service and
the GUI cannot fight over the session.

## Data and cache locations

- Session and message database: `~/.local/share/yaawp/`
- Downloaded media and avatars: `~/.cache/yaawp/`
- Runtime socket and lock: `$XDG_RUNTIME_DIR/yaawp/`

## Project layout

```
daemon/    Go service (whatsmeow engine, IPC server, session store)
gui/       C++ and QML Kirigami application
ipc/       protocol.md, the wire contract shared by both sides
packaging/ systemd user unit and a PKGBUILD
docs/      architecture and risk notes
install.sh local build and install helper
```

## Risks

yaawp is unofficial. Using it can lead to an account ban and it violates the
WhatsApp terms of service. Read docs/RISKS.md before linking a real account.

## Disclaimer

yaawp is not affiliated with, endorsed by, or connected to WhatsApp LLC or Meta
Platforms. WhatsApp is a trademark of its respective owners. This project is an
independent, unofficial client provided for research and personal use.

## License

No license has been chosen yet. Until one is added the project is provided as
is, for personal and research use, with all rights reserved by the author.
