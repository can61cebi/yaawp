# yaawp

A native WhatsApp client for Arch Linux and KDE Plasma. yaawp speaks the
WhatsApp multi-device protocol directly through a background service and
presents it in a native Kirigami interface. It is not a web wrapper and it does
not embed a browser.

## Status

Experimental. The project is an early skeleton. Login over QR, sending text,
and receiving live messages are the first targets. See docs/RISKS.md before
using it with a real account.

## Design

yaawp runs as two processes:

1. A headless Go daemon built on the whatsmeow library. It holds the session,
   performs the Noise handshake and Signal end to end encryption, and stores
   state in SQLite.
2. A native C++ and QML application built with Qt 6 and Kirigami. It talks to
   the daemon over a local Unix socket using a small newline-delimited JSON
   protocol.

The two processes keep the Go engine and the Qt event loop cleanly separated.
The daemon can run as a systemd user service and keep the session alive across
GUI restarts. See docs/ARCHITECTURE.md and ipc/protocol.md for details.

## Requirements

Runtime:

- qt6-base, qt6-declarative
- kirigami, prison, qqc2-desktop-style

Build:

- go
- cmake, extra-cmake-modules, ninja
- a C and C++ toolchain

On Arch Linux:

```
sudo pacman -S --needed go cmake extra-cmake-modules ninja \
    qt6-base qt6-declarative kirigami prison qqc2-desktop-style
```

## Build

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

## Run

Start the daemon in one terminal:

```
./daemon/bin/yaawp-daemon
```

Start the GUI in another terminal:

```
./gui/build/bin/yaawp
```

On first run the daemon is not paired. The GUI shows a QR code. Open WhatsApp
on your phone, go to Linked Devices, and scan it.

## Project layout

```
daemon/    Go service (whatsmeow engine, IPC server, session store)
gui/       C++ and QML Kirigami application
ipc/       protocol.md, the wire contract shared by both sides
packaging/ systemd user unit and a PKGBUILD draft
docs/      architecture and risk notes
```

## Risks

yaawp is unofficial. Using it can lead to an account ban and it violates the
WhatsApp terms of service. Read docs/RISKS.md.

## Disclaimer

yaawp is not affiliated with, endorsed by, or connected to WhatsApp LLC or Meta
Platforms. WhatsApp is a trademark of its respective owners. This project is an
independent, unofficial client provided for research and personal use.

## License

Not decided yet.
