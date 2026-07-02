#include "settings.h"

Settings::Settings(QObject *parent)
    : QObject(parent)
    , m_settings(QStringLiteral("cebi"), QStringLiteral("yaawp"))
    , m_rememberScroll(m_settings.value(QStringLiteral("rememberScroll"), true).toBool())
    , m_notifications(m_settings.value(QStringLiteral("notifications"), true).toBool())
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

void Settings::setMessageScale(double value)
{
    if (qFuzzyCompare(m_messageScale, value)) {
        return;
    }
    m_messageScale = value;
    m_settings.setValue(QStringLiteral("messageScale"), value);
    Q_EMIT messageScaleChanged();
}
