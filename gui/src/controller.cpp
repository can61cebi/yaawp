#include "controller.h"
#include "ipcclient.h"

#include <QClipboard>
#include <QDateTime>
#include <QDesktopServices>
#include <QDir>
#include <QGuiApplication>
#include <QImage>
#include <QStandardPaths>
#include <QUrl>

Controller::Controller(IpcClient *ipc, QObject *parent)
    : QObject(parent)
    , m_ipc(ipc)
{
    connect(ipc, &IpcClient::qrReceived, this, &Controller::onQrReceived);
    connect(ipc, &IpcClient::connectionStateChanged, this, &Controller::onConnectionStateChanged);
    connect(ipc, &IpcClient::pairSucceeded, this, &Controller::onPairSucceeded);
    connect(ipc, &IpcClient::eventReceived, this, &Controller::onEvent);
    connect(ipc, &IpcClient::chatPresenceChanged, this, &Controller::onChatPresence);
    connect(ipc, &IpcClient::presenceChanged, this, &Controller::onPresence);
    connect(ipc, &IpcClient::groupInfoReceived, this, &Controller::onGroupInfoReceived);
    connect(ipc, &IpcClient::starredReceived, this, &Controller::onStarredReceived);
}

void Controller::requestGroupInfo(const QString &jid)
{
    m_ipc->requestGroupInfo(jid);
}

void Controller::onGroupInfoReceived(const QJsonObject &info)
{
    m_groupInfo = info.toVariantMap();
    Q_EMIT groupInfoChanged();
}

void Controller::requestStarred()
{
    m_ipc->requestStarred();
}

void Controller::onStarredReceived(const QJsonArray &messages)
{
    m_starred = messages.toVariantList();
    Q_EMIT starredChanged();
}

void Controller::setCurrentChatJid(const QString &jid)
{
    if (m_currentChatJid == jid) {
        return;
    }
    m_currentChatJid = jid;
    m_typing = false;
    m_online = false;
    m_lastSeen = 0;
    updateStatus();
    Q_EMIT currentChatChanged();
    if (!jid.isEmpty()) {
        m_ipc->subscribePresence(jid);
    }
}

void Controller::onQrReceived(const QString &code)
{
    m_qrCode = code;
    Q_EMIT qrCodeChanged();
}

void Controller::onConnectionStateChanged(const QString &state)
{
    m_connectionState = state;
    Q_EMIT connectionStateChanged();
    if (state == QStringLiteral("connected")) {
        m_qrCode.clear();
        Q_EMIT qrCodeChanged();
        m_ipc->requestChats();
    }
}

void Controller::onPairSucceeded(const QString &jid, const QString &pushName)
{
    Q_UNUSED(jid)
    Q_UNUSED(pushName)
    // Pairing is done; a connection event follows and refreshes the UI.
}

void Controller::onEvent(const QString &event, const QJsonObject &data)
{
    Q_UNUSED(data)
    // History sync arrives in batches after linking; refresh the chat list so
    // conversations appear without the user pressing Refresh.
    if (event == QStringLiteral("history_sync")) {
        m_ipc->requestChats();
    }
}

void Controller::onChatPresence(const QString &chatJid, const QString &senderJid, const QString &state)
{
    Q_UNUSED(senderJid)
    if (chatJid != m_currentChatJid) {
        return;
    }
    m_typing = (state == QStringLiteral("composing"));
    updateStatus();
}

void Controller::onPresence(const QString &jid, const QString &state, qint64 lastSeen)
{
    if (jid != m_currentChatJid) {
        return;
    }
    m_online = (state == QStringLiteral("available"));
    if (lastSeen > 0) {
        m_lastSeen = lastSeen;
    }
    updateStatus();
}

void Controller::copyToClipboard(const QString &text) const
{
    if (QClipboard *clipboard = QGuiApplication::clipboard()) {
        clipboard->setText(text);
    }
}

void Controller::openFile(const QString &path) const
{
    if (!path.isEmpty()) {
        QDesktopServices::openUrl(QUrl::fromLocalFile(path));
    }
}

QString Controller::takeClipboardImage() const
{
    const QImage image = QGuiApplication::clipboard()->image();
    if (image.isNull()) {
        return QString();
    }
    const QString dir = QStandardPaths::writableLocation(QStandardPaths::TempLocation);
    const QString path = QDir(dir).filePath(
        QStringLiteral("yaawp-paste-%1.png").arg(QDateTime::currentMSecsSinceEpoch()));
    if (!image.save(path, "PNG")) {
        return QString();
    }
    return path;
}

void Controller::saveScroll(const QString &jid, double contentY)
{
    if (!jid.isEmpty()) {
        m_scroll.insert(jid, contentY);
    }
}

double Controller::savedScroll(const QString &jid) const
{
    return m_scroll.value(jid, -1.0);
}

void Controller::updateStatus()
{
    QString status;
    if (m_typing) {
        status = QStringLiteral("typing");
    } else if (m_online) {
        status = QStringLiteral("online");
    } else if (m_lastSeen > 0) {
        status = QStringLiteral("last seen ")
            + QDateTime::fromSecsSinceEpoch(m_lastSeen).toString(QStringLiteral("d MMM hh:mm"));
    }
    if (status != m_currentChatStatus) {
        m_currentChatStatus = status;
        Q_EMIT currentChatStatusChanged();
    }
}
