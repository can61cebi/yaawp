#include "settings.h"

Settings::Settings(QObject *parent)
    : QObject(parent)
    , m_settings(QStringLiteral("cebi"), QStringLiteral("yaawp"))
    , m_rememberScroll(m_settings.value(QStringLiteral("rememberScroll"), true).toBool())
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
