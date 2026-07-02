#include "notifier.h"
#include "controller.h"
#include "ipcclient.h"
#include "settings.h"

#include <KNotification>
#include <KNotificationReplyAction>

#include <memory>

Notifier::Notifier(IpcClient *ipc, Controller *controller, Settings *settings, QObject *parent)
    : QObject(parent)
    , m_ipc(ipc)
    , m_controller(controller)
    , m_settings(settings)
{
    connect(ipc, &IpcClient::messageReceived, this, &Notifier::onMessageReceived);
    connect(ipc, &IpcClient::chatsReceived, this, &Notifier::onChatsReceived);
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
    notification->setIconName(QStringLiteral("internet-mail"));

    auto reply = std::make_unique<KNotificationReplyAction>(QStringLiteral("Reply"));
    reply->setPlaceholderText(QStringLiteral("Reply to %1").arg(title));
    connect(reply.get(), &KNotificationReplyAction::replied, this, [this, chatJid](const QString &replyText) {
        m_ipc->sendText(chatJid, replyText);
    });
    notification->setReplyAction(std::move(reply));

    notification->sendEvent();
}
