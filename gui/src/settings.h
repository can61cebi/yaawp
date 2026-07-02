#pragma once

#include <QObject>
#include <QSettings>

// Settings holds persisted user preferences, exposed to QML.
class Settings : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool rememberScroll READ rememberScroll WRITE setRememberScroll NOTIFY rememberScrollChanged)
    Q_PROPERTY(bool notifications READ notifications WRITE setNotifications NOTIFY notificationsChanged)
    Q_PROPERTY(bool readReceipts READ readReceipts WRITE setReadReceipts NOTIFY readReceiptsChanged)
    Q_PROPERTY(bool shareOnline READ shareOnline WRITE setShareOnline NOTIFY shareOnlineChanged)
    Q_PROPERTY(double messageScale READ messageScale WRITE setMessageScale NOTIFY messageScaleChanged)

public:
    explicit Settings(QObject *parent = nullptr);

    bool rememberScroll() const { return m_rememberScroll; }
    void setRememberScroll(bool value);
    bool notifications() const { return m_notifications; }
    void setNotifications(bool value);
    bool readReceipts() const { return m_readReceipts; }
    void setReadReceipts(bool value);
    bool shareOnline() const { return m_shareOnline; }
    void setShareOnline(bool value);
    double messageScale() const { return m_messageScale; }
    void setMessageScale(double value);

Q_SIGNALS:
    void rememberScrollChanged();
    void notificationsChanged();
    void readReceiptsChanged();
    void shareOnlineChanged();
    void messageScaleChanged();

private:
    QSettings m_settings;
    bool m_rememberScroll;
    bool m_notifications;
    bool m_readReceipts;
    bool m_shareOnline;
    double m_messageScale;
};
