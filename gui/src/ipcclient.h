#pragma once

#include <QByteArray>
#include <QHash>
#include <QJsonArray>
#include <QJsonObject>
#include <QLocalSocket>
#include <QObject>
#include <QString>
#include <QStringList>
#include <QTimer>

// IpcClient owns the Unix socket connection to yaawp-daemon and translates the
// newline-delimited JSON protocol into Qt signals and invokable methods.
class IpcClient : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool connected READ isConnected NOTIFY connectedChanged)

public:
    explicit IpcClient(QObject *parent = nullptr);

    bool isConnected() const;

    Q_INVOKABLE void connectToDaemon();
    Q_INVOKABLE void login();
    Q_INVOKABLE void logout();
    Q_INVOKABLE void requestChats();
    Q_INVOKABLE void requestMessages(const QString &chatJid, int limit = 50);
    Q_INVOKABLE void sendText(const QString &chatJid, const QString &text, const QString &quotedId = QString(), const QString &quotedSender = QString(), const QString &quotedText = QString());
    Q_INVOKABLE void setTyping(const QString &chatJid, bool composing);
    Q_INVOKABLE void subscribePresence(const QString &jid);
    Q_INVOKABLE void deleteMessage(const QString &chatJid, const QString &id);
    Q_INVOKABLE void sendReaction(const QString &chatJid, const QString &messageId, const QString &senderJid, bool fromMe, const QString &emoji);
    Q_INVOKABLE void sendMedia(const QString &chatJid, const QString &filePath, const QString &caption);
    void markRead(const QString &chatJid, const QStringList &ids);

Q_SIGNALS:
    void connectedChanged();
    void qrReceived(const QString &code);
    void pairSucceeded(const QString &jid, const QString &pushName);
    void connectionStateChanged(const QString &state);
    void messageReceived(const QJsonObject &message);
    void receiptReceived(const QJsonObject &receipt);
    void chatsReceived(const QJsonArray &chats);
    void messagesReceived(const QJsonArray &messages);
    void chatPresenceChanged(const QString &chatJid, const QString &senderJid, const QString &state);
    void presenceChanged(const QString &jid, const QString &state, qint64 lastSeen);
    void messageStatusChanged(const QString &chatJid, const QStringList &ids, const QString &status);
    void messageMediaChanged(const QString &chatJid, const QString &id, const QString &mediaPath);
    void messageRevoked(const QString &chatJid, const QString &id);
    void reactionReceived(const QString &chatJid, const QString &messageId, const QString &senderJid, const QString &emoji, bool fromMe);
    void eventReceived(const QString &event, const QJsonObject &data);

private Q_SLOTS:
    void onReadyRead();
    void onSocketConnected();
    void onSocketDisconnected();
    void onSocketError();

private:
    QString socketPath() const;
    void send(const QString &method, const QJsonObject &params = {});
    void handleLine(const QByteArray &line);
    void ensureDaemonRunning();
    void handleEvent(const QString &event, const QJsonObject &data);
    void handleResponse(const QString &id, bool ok, const QJsonValue &result);

    QLocalSocket m_socket;
    QByteArray m_buffer;
    quint64 m_nextId = 1;
    QHash<QString, QString> m_pending; // request id -> method
    QTimer m_reconnectTimer;
    bool m_spawnedDaemon = false;
};
