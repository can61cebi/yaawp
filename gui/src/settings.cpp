#include "settings.h"

Settings::Settings(QObject *parent)
    : QObject(parent)
    , m_settings(QStringLiteral("cebi"), QStringLiteral("yaawp"))
    , m_rememberScroll(m_settings.value(QStringLiteral("rememberScroll"), true).toBool())
    , m_notifications(m_settings.value(QStringLiteral("notifications"), true).toBool())
    , m_readReceipts(m_settings.value(QStringLiteral("readReceipts"), true).toBool())
    , m_shareOnline(m_settings.value(QStringLiteral("shareOnline"), true).toBool())
    , m_messageScale(m_settings.value(QStringLiteral("messageScale"), 1.0).toDouble())
{
}

void Settings::setRememberScroll(bool value)
{
    if (m_rememberScroll == value) {
        return;
    }
    m_rememberScroll = value;
    m_settings.setValue(QStringLiteral("rememberScroll"), value);
    Q_EMIT rememberScrollChanged();
}

void Settings::setNotifications(bool value)
{
    if (m_notifications == value) {
        return;
    }
    m_notifications = value;
    m_settings.setValue(QStringLiteral("notifications"), value);
    Q_EMIT notificationsChanged();
}

void Settings::setReadReceipts(bool value)
{
    if (m_readReceipts == value) {
        return;
    }
    m_readReceipts = value;
    m_settings.setValue(QStringLiteral("readReceipts"), value);
    Q_EMIT readReceiptsChanged();
}

void Settings::setShareOnline(bool value)
{
    if (m_shareOnline == value) {
        return;
    }
    m_shareOnline = value;
    m_settings.setValue(QStringLiteral("shareOnline"), value);
    Q_EMIT shareOnlineChanged();
}

void Settings::setMessageScale(double value)
{
    if (qFuzzyCompare(m_messageScale, value)) {
        return;
    }
    m_messageScale = value;
    m_settings.setValue(QStringLiteral("messageScale"), value);
    Q_EMIT messageScaleChanged();
}
