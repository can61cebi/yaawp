#pragma once

#include <QObject>
#include <QSettings>

// Settings holds persisted user preferences, exposed to QML.
class Settings : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool rememberScroll READ rememberScroll WRITE setRememberScroll NOTIFY rememberScrollChanged)

public:
    explicit Settings(QObject *parent = nullptr);

    bool rememberScroll() const { return m_rememberScroll; }
    void setRememberScroll(bool value);

Q_SIGNALS:
    void rememberScrollChanged();

private:
    QSettings m_settings;
    bool m_rememberScroll;
};
