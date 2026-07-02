# Roadmap

Status legend: [x] done, [~] in progress, [ ] planned.

## Milestone 1: usable one to one messaging

- [x] Two process architecture, IPC contract, native Kirigami GUI
- [x] QR login, pairing, live send and receive of text
- [x] Local SQLite store for chats and messages
- [x] History sync persistence and contact name resolution
- [x] Native notifications for incoming messages
- [x] LID to phone number normalization
- [x] Media and non text messages shown as placeholders
- [x] Browser like device identity in Linked Devices
- [x] Chat bubble layout fix
- [x] Deliver the current QR to a client that connects after it was generated
- [x] Read receipts: mark incoming messages read when a chat is open
- [x] Deduplicate the local echo of a sent message against the stored copy
- [x] Chat list preview update and move to top on new activity
- [~] Reconnect handling and a visible connection status in the header

## Milestone 2: richer conversation view

- [x] Message timestamps and day separators
- [x] Sender names and colors inside group chats
- [x] Delivery and read state ticks on outgoing messages
- [x] Presence in the header: online, last seen, typing
- [x] Message reactions: send, receive, and display
- [x] Message context menu: copy and delete for everyone
- [ ] Reply and quote
- [ ] Unread counts and ordering in the chat list

## Milestone 3: media

- [x] Download and show images and stickers inline
- [ ] Voice message playback
- [ ] Video and document open with the system handler
- [ ] Send images and files from the GUI
- [ ] On demand download for history media
- [ ] Media cache size limits

## Milestone 4: KDE integration

- [ ] System tray icon with an unread badge
- [ ] Inline reply action from a notification
- [ ] Store session secrets in KWallet
- [ ] systemd user service for the daemon with autostart
- [ ] Install target and application menu entry

## Milestone 5: distribution and polish

- [ ] Finalize the PKGBUILD and publish to the AUR
- [ ] Search across chats and messages
- [ ] Settings page: notifications, device name, theme
- [ ] Multiple accounts
- [ ] Basic test coverage for the store and the IPC layer

## Known limitations

- Group read receipts need a per message participant and are not sent yet.
- Historical duplicates from before LID normalization are resolved by a clean
  re-link. A migration to merge them in place is not implemented.
- The device name is fixed at pair time, so changing it needs a re-link.
- History media is not downloaded automatically; only live media is fetched.
