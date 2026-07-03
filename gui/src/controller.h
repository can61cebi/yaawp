#pragma once

#include <QHash>
#include <QJsonArray>
#include <QJsonObject>
#include <QObject>
#include <QString>
#include <QVariantList>
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
    Q_PROPERTY(QVariantMap contactInfo READ contactInfo NOTIFY contactInfoChanged)
    Q_PROPERTY(QVariantList starredMessages READ starredMessages NOTIFY starredChanged)
    // True when the app was launched with --hidden, so the window stays in the
    // tray at startup. Set once before the QML loads.
    Q_PROPERTY(bool startHidden READ startHidden NOTIFY startHiddenChanged)
    // Whether a desktop autostart entry exists, so the app opens on login.
    Q_PROPERTY(bool autostartEnabled READ autostartEnabled WRITE setAutostartEnabled NOTIFY autostartEnabledChanged)

public:
    explicit Controller(IpcClient *ipc, QObject *parent = nullptr);

    QString connectionState() const { return m_connectionState; }
    QString qrCode() const { return m_qrCode; }
    bool loggedIn() const { return m_connectionState == QStringLiteral("connected"); }
    QString currentChatJid() const { return m_currentChatJid; }
    QString currentChatStatus() const { return m_currentChatStatus; }
    QVariantMap groupInfo() const { return m_groupInfo; }
    QVariantMap contactInfo() const { return m_contactInfo; }
    QVariantList starredMessages() const { return m_starred; }
    void setCurrentChatJid(const QString &jid);

    bool startHidden() const { return m_startHidden; }
    void setStartHidden(bool hidden);
    bool autostartEnabled() const;
    void setAutostartEnabled(bool enabled);

    Q_INVOKABLE void copyToClipboard(const QString &text) const;
    Q_INVOKABLE void openFile(const QString &path) const;
    // If the clipboard holds an image, save it to a temp file and return the
    // path, otherwise return an empty string.
    Q_INVOKABLE QString takeClipboardImage() const;
    Q_INVOKABLE void openUrl(const QString &url) const;
    // Return a fresh temp file url for a voice recording.
    Q_INVOKABLE QString newVoiceFile() const;
    Q_INVOKABLE void saveScroll(const QString &jid, double contentY);
    Q_INVOKABLE double savedScroll(const QString &jid) const;
    Q_INVOKABLE void requestGroupInfo(const QString &jid);
    Q_INVOKABLE void requestContactInfo(const QString &jid);
    Q_INVOKABLE void requestStarred();

Q_SIGNALS:
    void connectionStateChanged();
    void qrCodeChanged();
    void currentChatChanged();
    void currentChatStatusChanged();
    void groupInfoChanged();
    void contactInfoChanged();
    void starredChanged();
    void startHiddenChanged();
    void autostartEnabledChanged();

private Q_SLOTS:
    void onQrReceived(const QString &code);
    void onConnectionStateChanged(const QString &state);
    void onPairSucceeded(const QString &jid, const QString &pushName);
    void onEvent(const QString &event, const QJsonObject &data);
    void onChatPresence(const QString &chatJid, const QString &senderJid, const QString &state);
    void onPresence(const QString &jid, const QString &state, qint64 lastSeen);
    void onGroupInfoReceived(const QJsonObject &info);
    void onContactInfoReceived(const QJsonObject &info);
    void onStarredReceived(const QJsonArray &messages);

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
    QVariantMap m_contactInfo;
    QVariantList m_starred;
    bool m_startHidden = false;

    QString autostartFilePath() const;
};
