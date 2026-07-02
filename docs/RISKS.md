# Risks

yaawp is an experimental, unofficial client. The following risks are accepted
for this project.

1. Account ban. Meta detects and can ban accounts that use unofficial clients.
   whatsmeow tries to mimic the official client, but there is no guarantee.
   This is the largest risk.
2. Terms of service. Using an unofficial client violates the WhatsApp terms of
   service.
3. Protocol churn. WhatsApp changes the protocol over time. The engine needs
   updates and features break periodically.
4. Companion device requirement. The first link needs the primary phone to
   scan a QR code or enter a pairing code. After that the companion can operate
   while the phone is offline, but the account still belongs to the phone.
5. End to end encryption correctness. Getting encryption wrong produces
   undelivered or undecryptable messages. Relying on whatsmeow reduces this
   risk.

## Mitigations

- Keep the whatsmeow dependency current.
- Test protocol changes against a controlled account before wider use.
- Never log message plaintext or session secrets.
