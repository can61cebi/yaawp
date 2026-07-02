#pragma once

#include <QObject>
#include <QString>

class IpcClient;

// Controller exposes login and connection state to QML and tracks the open chat.
class Controller : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString connectionState READ connectionState NOTIFY connectionStateChanged)
    Q_PROPERTY(QString qrCode READ qrCode NOTIFY qrCodeChanged)
    Q_PROPERTY(bool loggedIn READ loggedIn NOTIFY connectionStateChanged)
    Q_PROPERTY(QString currentChatJid READ currentChatJid WRITE setCurrentChatJid NOTIFY currentChatChanged)

public:
    explicit Controller(IpcClient *ipc, QObject *parent = nullptr);

    QString connectionState() const { return m_connectionState; }
    QString qrCode() const { return m_qrCode; }
    bool loggedIn() const { return m_connectionState == QStringLiteral("connected"); }
    QString currentChatJid() const { return m_currentChatJid; }
    void setCurrentChatJid(const QString &jid);

Q_SIGNALS:
    void connectionStateChanged();
    void qrCodeChanged();
    void currentChatChanged();

private Q_SLOTS:
    void onQrReceived(const QString &code);
    void onConnectionStateChanged(const QString &state);
    void onPairSucceeded(const QString &jid, const QString &pushName);

private:
    IpcClient *m_ipc;
    QString m_connectionState = QStringLiteral("logged_out");
    QString m_qrCode;
    QString m_currentChatJid;
};
