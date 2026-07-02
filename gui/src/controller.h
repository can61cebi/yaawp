#pragma once

#include <QHash>
#include <QJsonObject>
#include <QObject>
#include <QString>
#include <QVariantMap>

class IpcClient;

// Controller exposes login and connection state to QML, tracks the open chat,
// and derives a human readable status line (online, typing, last seen) for it.
class Controller : public QObject
{
    Q_OBJECT
    Q_PROPERTY(QString connectionState READ connectionState NOTIFY connectionStateChanged)
    Q_PROPERTY(QString qrCode READ qrCode NOTIFY qrCodeChanged)
    Q_PROPERTY(bool loggedIn READ loggedIn NOTIFY connectionStateChanged)
    Q_PROPERTY(QString currentChatJid READ currentChatJid WRITE setCurrentChatJid NOTIFY currentChatChanged)
    Q_PROPERTY(QString currentChatStatus READ currentChatStatus NOTIFY currentChatStatusChanged)
    Q_PROPERTY(QVariantMap groupInfo READ groupInfo NOTIFY groupInfoChanged)

public:
    explicit Controller(IpcClient *ipc, QObject *parent = nullptr);

    QString connectionState() const { return m_connectionState; }
    QString qrCode() const { return m_qrCode; }
    bool loggedIn() const { return m_connectionState == QStringLiteral("connected"); }
    QString currentChatJid() const { return m_currentChatJid; }
    QString currentChatStatus() const { return m_currentChatStatus; }
    QVariantMap groupInfo() const { return m_groupInfo; }
    void setCurrentChatJid(const QString &jid);

    Q_INVOKABLE void copyToClipboard(const QString &text) const;
    Q_INVOKABLE void openFile(const QString &path) const;
    Q_INVOKABLE void saveScroll(const QString &jid, double contentY);
    Q_INVOKABLE double savedScroll(const QString &jid) const;
    Q_INVOKABLE void requestGroupInfo(const QString &jid);

Q_SIGNALS:
    void connectionStateChanged();
    void qrCodeChanged();
    void currentChatChanged();
    void currentChatStatusChanged();
    void groupInfoChanged();

private Q_SLOTS:
    void onQrReceived(const QString &code);
    void onConnectionStateChanged(const QString &state);
    void onPairSucceeded(const QString &jid, const QString &pushName);
    void onEvent(const QString &event, const QJsonObject &data);
    void onChatPresence(const QString &chatJid, const QString &senderJid, const QString &state);
    void onPresence(const QString &jid, const QString &state, qint64 lastSeen);
    void onGroupInfoReceived(const QJsonObject &info);

private:
    void updateStatus();

    IpcClient *m_ipc;
    QString m_connectionState = QStringLiteral("logged_out");
    QString m_qrCode;
    QString m_currentChatJid;
    QString m_currentChatStatus;
    bool m_typing = false;
    bool m_online = false;
    qint64 m_lastSeen = 0;
    QHash<QString, double> m_scroll; // chat jid -> saved content y offset
    QVariantMap m_groupInfo;
};
