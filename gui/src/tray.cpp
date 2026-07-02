#include "tray.h"

#include "ipcclient.h"

#include <KStatusNotifierItem>

#include <QString>
#include <QWindow>

#include <utility>

Tray::Tray(IpcClient *ipc, QObject *parent)
    : QObject(parent)
    , m_sni(new KStatusNotifierItem(QStringLiteral("tr.cebi.yaawp"), this))
{
    m_sni->setIconByName(QStringLiteral("internet-mail"));
    m_sni->setCategory(KStatusNotifierItem::Communications);
    m_sni->setStatus(KStatusNotifierItem::Active);
    m_sni->setTitle(QStringLiteral("yaawp"));
    m_sni->setToolTip(QStringLiteral("internet-mail"), QStringLiteral("yaawp"), QString());
    m_sni->setStandardActionsEnabled(true);

    connect(ipc, &IpcClient::chatUnreadChanged, this, &Tray::onChatUnread);
}

void Tray::setWindow(QWindow *window)
{
    m_sni->setAssociatedWindow(window);
}

void Tray::onChatUnread(const QString &jid, int unread)
{
    if (unread > 0) {
        m_unread.insert(jid, unread);
    } else {
        m_unread.remove(jid);
    }
    refreshTooltip();
}

void Tray::refreshTooltip()
{
    int total = 0;
    for (const int n : std::as_const(m_unread)) {
        total += n;
    }
    if (total > 0) {
        m_sni->setToolTip(QStringLiteral("internet-mail"), QStringLiteral("yaawp"),
                          QStringLiteral("%1 unread").arg(total));
        m_sni->setStatus(KStatusNotifierItem::NeedsAttention);
    } else {
        m_sni->setToolTip(QStringLiteral("internet-mail"), QStringLiteral("yaawp"), QString());
        m_sni->setStatus(KStatusNotifierItem::Active);
    }
}
