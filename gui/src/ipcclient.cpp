#include "ipcclient.h"

#include <QCoreApplication>
#include <QDir>
#include <QFileInfo>
#include <QJsonDocument>
#include <QProcess>
#include <QProcessEnvironment>
#include <QStandardPaths>

IpcClient::IpcClient(QObject *parent)
    : QObject(parent)
{
    connect(&m_socket, &QLocalSocket::readyRead, this, &IpcClient::onReadyRead);
    connect(&m_socket, &QLocalSocket::connected, this, &IpcClient::onSocketConnected);
    connect(&m_socket, &QLocalSocket::disconnected, this, &IpcClient::onSocketDisconnected);
    connect(&m_socket, &QLocalSocket::errorOccurred, this, &IpcClient::onSocketError);

    m_reconnectTimer.setInterval(2000);
    m_reconnectTimer.setSingleShot(true);
    connect(&m_reconnectTimer, &QTimer::timeout, this, &IpcClient::connectToDaemon);
}

QString IpcClient::socketPath() const
{
    QString base = QProcessEnvironment::systemEnvironment().value(QStringLiteral("XDG_RUNTIME_DIR"));
    if (base.isEmpty()) {
        base = QDir::tempPath();
    }
    return base + QStringLiteral("/yaawp/daemon.sock");
}

bool IpcClient::isConnected() const
{
    return m_socket.state() == QLocalSocket::ConnectedState;
}

void IpcClient::connectToDaemon()
{
    if (m_socket.state() != QLocalSocket::UnconnectedState) {
        return;
    }
    m_socket.connectToServer(socketPath());
}

void IpcClient::onSocketConnected()
{
    Q_EMIT connectedChanged();
    send(QStringLiteral("get_state"));
}

void IpcClient::onSocketDisconnected()
{
    Q_EMIT connectedChanged();
    m_reconnectTimer.start();
}

void IpcClient::onSocketError()
{
    Q_EMIT connectedChanged();
    ensureDaemonRunning();
    if (!m_reconnectTimer.isActive()) {
        m_reconnectTimer.start();
    }
}

void IpcClient::ensureDaemonRunning()
{
    if (m_spawnedDaemon) {
        return;
    }
    QString exe = QStandardPaths::findExecutable(QStringLiteral("yaawp-daemon"));
    if (exe.isEmpty()) {
        // Development fallback relative to the GUI binary (gui/build/bin/yaawp).
        const QString dev = QCoreApplication::applicationDirPath()
            + QStringLiteral("/../../../daemon/bin/yaawp-daemon");
        if (QFileInfo::exists(dev)) {
            exe = dev;
        }
    }
    if (!exe.isEmpty()) {
        m_spawnedDaemon = QProcess::startDetached(exe, {});
    }
}

void IpcClient::onReadyRead()
{
    m_buffer.append(m_socket.readAll());
    int idx;
    while ((idx = m_buffer.indexOf('\n')) != -1) {
        const QByteArray line = m_buffer.left(idx);
        m_buffer.remove(0, idx + 1);
        if (!line.trimmed().isEmpty()) {
            handleLine(line);
        }
    }
}

void IpcClient::handleLine(const QByteArray &line)
{
    const QJsonDocument doc = QJsonDocument::fromJson(line);
    if (!doc.isObject()) {
        return;
    }
    const QJsonObject obj = doc.object();
    const QString type = obj.value(QStringLiteral("type")).toString();

    if (type == QStringLiteral("event")) {
        handleEvent(obj.value(QStringLiteral("event")).toString(),
                    obj.value(QStringLiteral("data")).toObject());
    } else if (type == QStringLiteral("resp")) {
        handleResponse(obj.value(QStringLiteral("id")).toString(),
                       obj.value(QStringLiteral("ok")).toBool(),
                       obj.value(QStringLiteral("result")));
    }
}

void IpcClient::handleEvent(const QString &event, const QJsonObject &data)
{
    if (event == QStringLiteral("qr")) {
        Q_EMIT qrReceived(data.value(QStringLiteral("code")).toString());
    } else if (event == QStringLiteral("pair_success")) {
        Q_EMIT pairSucceeded(data.value(QStringLiteral("jid")).toString(),
                             data.value(QStringLiteral("push_name")).toString());
    } else if (event == QStringLiteral("connection")) {
        Q_EMIT connectionStateChanged(data.value(QStringLiteral("state")).toString());
    } else if (event == QStringLiteral("message")) {
        Q_EMIT messageReceived(data);
    } else if (event == QStringLiteral("receipt")) {
        Q_EMIT receiptReceived(data);
    } else if (event == QStringLiteral("chat_presence")) {
        Q_EMIT chatPresenceChanged(data.value(QStringLiteral("chat_jid")).toString(),
                                   data.value(QStringLiteral("sender_jid")).toString(),
                                   data.value(QStringLiteral("state")).toString());
    } else if (event == QStringLiteral("presence")) {
        Q_EMIT presenceChanged(data.value(QStringLiteral("jid")).toString(),
                               data.value(QStringLiteral("state")).toString(),
                               static_cast<qint64>(data.value(QStringLiteral("last_seen")).toDouble()));
    } else if (event == QStringLiteral("message_status")) {
        const QJsonArray idsArray = data.value(QStringLiteral("message_ids")).toArray();
        QStringList ids;
        for (const QJsonValue &value : idsArray) {
            ids.append(value.toString());
        }
        Q_EMIT messageStatusChanged(data.value(QStringLiteral("chat_jid")).toString(), ids,
                                    data.value(QStringLiteral("status")).toString());
    } else if (event == QStringLiteral("message_media")) {
        Q_EMIT messageMediaChanged(data.value(QStringLiteral("chat_jid")).toString(),
                                   data.value(QStringLiteral("id")).toString(),
                                   data.value(QStringLiteral("media_path")).toString());
    } else if (event == QStringLiteral("message_revoked")) {
        Q_EMIT messageRevoked(data.value(QStringLiteral("chat_jid")).toString(),
                              data.value(QStringLiteral("message_id")).toString());
    } else if (event == QStringLiteral("reaction")) {
        Q_EMIT reactionReceived(data.value(QStringLiteral("chat_jid")).toString(),
                                data.value(QStringLiteral("message_id")).toString(),
                                data.value(QStringLiteral("sender_jid")).toString(),
                                data.value(QStringLiteral("emoji")).toString(),
                                data.value(QStringLiteral("from_me")).toBool());
    }
    Q_EMIT eventReceived(event, data);
}

void IpcClient::handleResponse(const QString &id, bool ok, const QJsonValue &result)
{
    const QString method = m_pending.take(id);
    if (!ok) {
        return;
    }
    if (method == QStringLiteral("list_chats")) {
        Q_EMIT chatsReceived(result.toArray());
    } else if (method == QStringLiteral("list_messages")) {
        Q_EMIT messagesReceived(result.toArray());
    }
}

void IpcClient::send(const QString &method, const QJsonObject &params)
{
    if (m_socket.state() != QLocalSocket::ConnectedState) {
        return;
    }
    const QString id = QString::number(m_nextId++);
    m_pending.insert(id, method);

    QJsonObject cmd;
    cmd.insert(QStringLiteral("type"), QStringLiteral("cmd"));
    cmd.insert(QStringLiteral("id"), id);
    cmd.insert(QStringLiteral("method"), method);
    if (!params.isEmpty()) {
        cmd.insert(QStringLiteral("params"), params);
    }
    QByteArray payload = QJsonDocument(cmd).toJson(QJsonDocument::Compact);
    payload.append('\n');
    m_socket.write(payload);
}

void IpcClient::login()
{
    send(QStringLiteral("login"));
}

void IpcClient::logout()
{
    send(QStringLiteral("logout"));
}

void IpcClient::requestChats()
{
    send(QStringLiteral("list_chats"));
}

void IpcClient::requestMessages(const QString &chatJid, int limit)
{
    QJsonObject p;
    p.insert(QStringLiteral("chat_jid"), chatJid);
    p.insert(QStringLiteral("limit"), limit);
    send(QStringLiteral("list_messages"), p);
}

void IpcClient::sendText(const QString &chatJid, const QString &text)
{
    QJsonObject p;
    p.insert(QStringLiteral("chat_jid"), chatJid);
    p.insert(QStringLiteral("text"), text);
    send(QStringLiteral("send_text"), p);
}

void IpcClient::setTyping(const QString &chatJid, bool composing)
{
    QJsonObject p;
    p.insert(QStringLiteral("chat_jid"), chatJid);
    p.insert(QStringLiteral("composing"), composing);
    send(QStringLiteral("set_typing"), p);
}

void IpcClient::subscribePresence(const QString &jid)
{
    QJsonObject p;
    p.insert(QStringLiteral("jid"), jid);
    send(QStringLiteral("subscribe_presence"), p);
}

void IpcClient::deleteMessage(const QString &chatJid, const QString &id)
{
    QJsonObject p;
    p.insert(QStringLiteral("chat_jid"), chatJid);
    p.insert(QStringLiteral("message_id"), id);
    send(QStringLiteral("delete_message"), p);
}

void IpcClient::sendReaction(const QString &chatJid, const QString &messageId, const QString &senderJid, bool fromMe, const QString &emoji)
{
    QJsonObject p;
    p.insert(QStringLiteral("chat_jid"), chatJid);
    p.insert(QStringLiteral("message_id"), messageId);
    p.insert(QStringLiteral("sender_jid"), senderJid);
    p.insert(QStringLiteral("from_me"), fromMe);
    p.insert(QStringLiteral("emoji"), emoji);
    send(QStringLiteral("send_reaction"), p);
}

void IpcClient::markRead(const QString &chatJid, const QStringList &ids)
{
    if (ids.isEmpty()) {
        return;
    }
    QJsonArray arr;
    for (const QString &id : ids) {
        arr.append(id);
    }
    QJsonObject p;
    p.insert(QStringLiteral("chat_jid"), chatJid);
    p.insert(QStringLiteral("message_ids"), arr);
    send(QStringLiteral("mark_read"), p);
}
