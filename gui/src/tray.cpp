#include "tray.h"

#include <KStatusNotifierItem>

#include <QString>
#include <QWindow>

Tray::Tray(QObject *parent)
    : QObject(parent)
    , m_sni(new KStatusNotifierItem(QStringLiteral("tr.cebi.yaawp"), this))
{
    m_sni->setIconByName(QStringLiteral("internet-mail"));
    m_sni->setCategory(KStatusNotifierItem::Communications);
    m_sni->setStatus(KStatusNotifierItem::Active);
    m_sni->setTitle(QStringLiteral("yaawp"));
    m_sni->setToolTip(QStringLiteral("internet-mail"), QStringLiteral("yaawp"), QString());
    m_sni->setStandardActionsEnabled(true);
}

void Tray::setWindow(QWindow *window)
{
    m_sni->setAssociatedWindow(window);
}
