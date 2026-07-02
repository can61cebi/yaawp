#include "notifier.h"
#include "controller.h"
#include "ipcclient.h"

#include <KNotification>

Notifier::Notifier(IpcClient *ipc, Controller *controller, QObject *parent)
    : QObject(parent)
    , m_ipc(ipc)
    , m_controller(controller)
{
    connect(ipc, &IpcClient::messageReceived, this, &Notifier::onMessageReceived);
}

void Notifier::onMessageReceived(const QJsonObject &message)
{
    if (message.value(QStringLiteral("from_me")).toBool()) {
        return;
    }
    const QString chatJid = message.value(QStringLiteral("chat_jid")).toString();
    if (chatJid == m_controller->currentChatJid()) {
        return; // The user is already looking at this chat.
    }
    const QString text = message.value(QStringLiteral("text")).toString();
    if (text.isEmpty()) {
        return;
    }

    QString title = chatJid;
    const int at = title.indexOf(QLatin1Char('@'));
    if (at > 0) {
        title = title.left(at);
    }

    auto *notification = new KNotification(QStringLiteral("newMessage"));
    notification->setComponentName(QStringLiteral("yaawp"));
    notification->setTitle(title);
    notification->setText(text);
    notification->setIconName(QStringLiteral("internet-mail"));
    notification->sendEvent();
}
