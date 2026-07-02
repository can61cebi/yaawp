#pragma once

#include <QObject>
#include <QSettings>

// Settings holds persisted user preferences, exposed to QML.
class Settings : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool rememberScroll READ rememberScroll WRITE setRememberScroll NOTIFY rememberScrollChanged)
    Q_PROPERTY(bool notifications READ notifications WRITE setNotifications NOTIFY notificationsChanged)
    Q_PROPERTY(double messageScale READ messageScale WRITE setMessageScale NOTIFY messageScaleChanged)

public:
    explicit Settings(QObject *parent = nullptr);

    bool rememberScroll() const { return m_rememberScroll; }
    void setRememberScroll(bool value);
    bool notifications() const { return m_notifications; }
    void setNotifications(bool value);
    double messageScale() const { return m_messageScale; }
    void setMessageScale(double value);

Q_SIGNALS:
    void rememberScrollChanged();
    void notificationsChanged();
    void messageScaleChanged();

private:
    QSettings m_settings;
    bool m_rememberScroll;
    bool m_notifications;
    double m_messageScale;
};
