#pragma once

#include <QAbstractListModel>
#include <QJsonArray>
#include <QJsonObject>
#include <QList>
#include <QMap>
#include <QString>
#include <QStringList>

class IpcClient;

struct MessageItem {
    QString id;
    QString senderJid;
    QString senderName;
    bool fromMe = false;
    qint64 timestamp = 0;
    QString text;
    QString type;
    QString status;
    QString mediaPath;
    QString quotedText;
    QString quotedSender;
    QMap<QString, QString> reactions; // sender jid -> emoji
    bool pending = false; // local echo awaiting the server copy
};

// MessageModel holds the messages of the currently open chat. History is loaded
// from the daemon list_messages response; new messages arrive as events.
class MessageModel : public QAbstractListModel
{
    Q_OBJECT
    Q_PROPERTY(QString replyText READ replyText NOTIFY replyChanged)
    Q_PROPERTY(QString replySenderName READ replySenderName NOTIFY replyChanged)
    Q_PROPERTY(bool hasReply READ hasReply NOTIFY replyChanged)

public:
    enum Roles {
        IdRole = Qt::UserRole + 1,
        SenderRole,
        SenderNameRole,
        FromMeRole,
        TimestampRole,
        TextRole,
        TypeRole,
        DayRole,
        StatusRole,
        MediaPathRole,
        ReactionsRole,
        QuotedTextRole,
    };

    explicit MessageModel(IpcClient *ipc, QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = {}) const override;
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

    Q_INVOKABLE void setChat(const QString &jid);
    Q_INVOKABLE void sendText(const QString &text);
    Q_INVOKABLE void deleteMessage(const QString &id);
    Q_INVOKABLE void react(const QString &messageId, const QString &emoji);
    Q_INVOKABLE void setReplyTo(const QString &messageId);
    Q_INVOKABLE void clearReply();

    QString replyText() const { return m_replyText; }
    QString replySenderName() const { return m_replySenderName; }
    bool hasReply() const { return !m_replyId.isEmpty(); }

Q_SIGNALS:
    void replyChanged();

private Q_SLOTS:
    void onMessagesReceived(const QJsonArray &messages);
    void onMessageReceived(const QJsonObject &message);
    void onMessageStatus(const QString &chatJid, const QStringList &ids, const QString &status);
    void onMessageMedia(const QString &chatJid, const QString &id, const QString &mediaPath);
    void onMessageRevoked(const QString &chatJid, const QString &id);
    void onReaction(const QString &chatJid, const QString &messageId, const QString &senderJid, const QString &emoji, bool fromMe);

private:
    void append(const MessageItem &item);
    MessageItem fromJson(const QJsonObject &o) const;
    QString dayLabel(qint64 timestamp) const;

    IpcClient *m_ipc;
    QString m_chatJid;
    QList<MessageItem> m_messages;
    QString m_replyId;
    QString m_replySender;
    QString m_replyText;
    QString m_replySenderName;
};
