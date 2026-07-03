#include "notifier.h"
#include "controller.h"
#include "ipcclient.h"
#include "settings.h"

#include <KNotification>
#include <KNotificationReplyAction>

#include <QDir>
#include <QFile>
#include <QStandardPaths>

#include <memory>

Notifier::Notifier(IpcClient *ipc, Controller *controller, Settings *settings, QObject *parent)
    : QObject(parent)
    , m_ipc(ipc)
    , m_controller(controller)
    , m_settings(settings)
    , m_iconPath(ensureIconPath())
{
    connect(ipc, &IpcClient::messageReceived, this, &Notifier::onMessageReceived);
    connect(ipc, &IpcClient::chatsReceived, this, &Notifier::onChatsReceived);
}

QString Notifier::ensureIconPath()
{
    const QString dir = QStandardPaths::writableLocation(QStandardPaths::GenericCacheLocation)
        + QStringLiteral("/yaawp");
    QDir().mkpath(dir);
    const QString path = dir + QStringLiteral("/tr.cebi.yaawp.svg");
    // Refresh the copy each start so it tracks the bundled icon across updates.
    QFile::remove(path);
    if (QFile::copy(QStringLiteral(":/icons/tr.cebi.yaawp.svg"), path)) {
        QFile::setPermissions(path, QFileDevice::ReadOwner | QFileDevice::WriteOwner
                              | QFileDevice::ReadGroup | QFileDevice::ReadOther);
        return path;
    }
    // Fall back to the themed name if the resource could not be written out.
    return QStringLiteral("tr.cebi.yaawp");
}

void Notifier::onChatsReceived(const QJsonArray &chats)
{
    m_muted.clear();
    for (const QJsonValue &value : chats) {
        const QJsonObject o = value.toObject();
        if (o.value(QStringLiteral("muted")).toBool()) {
            m_muted.insert(o.value(QStringLiteral("jid")).toString());
        }
    }
}

void Notifier::onMessageReceived(const QJsonObject &message)
{
    if (message.value(QStringLiteral("from_me")).toBool()) {
        return;
    }
    if (!m_settings->notifications()) {
        return;
    }
    const QString chatJid = message.value(QStringLiteral("chat_jid")).toString();
    if (chatJid == m_controller->currentChatJid()) {
        return; // The user is already looking at this chat.
    }
    if (m_muted.contains(chatJid)) {
        return; // The chat is muted.
    }
    const QString text = message.value(QStringLiteral("text")).toString();
    if (text.isEmpty()) {
        return;
    }

    QString title = message.value(QStringLiteral("sender_name")).toString();
    if (title.isEmpty()) {
        title = chatJid;
        const int at = title.indexOf(QLatin1Char('@'));
        if (at > 0) {
            title = title.left(at);
        }
    }

    auto *notification = new KNotification(QStringLiteral("newMessage"));
    notification->setComponentName(QStringLiteral("yaawp"));
    notification->setTitle(title);
    notification->setText(text);
    notification->setIconName(m_iconPath);

    auto reply = std::make_unique<KNotificationReplyAction>(QStringLiteral("Reply"));
    reply->setPlaceholderText(QStringLiteral("Reply to %1").arg(title));
    connect(reply.get(), &KNotificationReplyAction::replied, this, [this, chatJid](const QString &replyText) {
        m_ipc->sendText(chatJid, replyText);
    });
    notification->setReplyAction(std::move(reply));

    notification->sendEvent();
}
