#include "tray.h"

#include "ipcclient.h"

#include <KStatusNotifierItem>

#include <QIcon>
#include <QString>
#include <QWindow>

#include <utility>

namespace {
// Resolve the app icon from the current theme, falling back to the copy compiled
// into the binary. A freshly installed hicolor icon is not always visible to an
// already-running plasmashell, so a bundled fallback keeps the tray from being
// blank.
QIcon loadAppIcon()
{
    QIcon icon = QIcon::fromTheme(QStringLiteral("tr.cebi.yaawp"));
    if (icon.isNull()) {
        icon = QIcon(QStringLiteral(":/icons/tr.cebi.yaawp.svg"));
    }
    return icon;
}
}

Tray::Tray(IpcClient *ipc, QObject *parent)
    : QObject(parent)
    , m_sni(new KStatusNotifierItem(QStringLiteral("tr.cebi.yaawp"), this))
    , m_icon(loadAppIcon())
{
    // Send the icon as pixmap data rather than by name. plasmashell renders the
    // bytes directly, so the tray icon shows reliably regardless of whether the
    // running shell has picked up the installed hicolor icon yet.
    m_sni->setIconByPixmap(m_icon);
    m_sni->setCategory(KStatusNotifierItem::Communications);
    m_sni->setStatus(KStatusNotifierItem::Active);
    m_sni->setTitle(QStringLiteral("yaawp"));
    m_sni->setToolTip(m_icon, QStringLiteral("yaawp"), QString());
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
        m_sni->setToolTip(m_icon, QStringLiteral("yaawp"),
                          QStringLiteral("%1 unread").arg(total));
        m_sni->setStatus(KStatusNotifierItem::NeedsAttention);
    } else {
        m_sni->setToolTip(m_icon, QStringLiteral("yaawp"), QString());
        m_sni->setStatus(KStatusNotifierItem::Active);
    }
}
