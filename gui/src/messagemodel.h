#pragma once

#include <QAbstractListModel>
#include <QJsonArray>
#include <QJsonObject>
#include <QList>
#include <QMap>
#include <QSet>
#include <QString>
#include <QStringList>
#include <QUrl>

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
    int mediaWidth = 0;
    int mediaHeight = 0;
    QString quotedId;
    QString quotedText;
    QString quotedSender;
    QString previewUrl;
    QString previewTitle;
    QString previewDesc;
    QString previewImage;
    QMap<QString, QString> reactions; // sender jid -> emoji
    bool edited = false;  // the sender edited this message
    bool starred = false; // the user starred this message
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
        DaySeparatorRole,
        StatusRole,
        MediaPathRole,
        MediaWidthRole,
        MediaHeightRole,
        ReactionsRole,
        QuotedTextRole,
        QuotedIdRole,
        EditedRole,
        StarredRole,
        PreviewUrlRole,
        PreviewTitleRole,
        PreviewDescRole,
        PreviewImageRole,
    };

    explicit MessageModel(IpcClient *ipc, QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = {}) const override;
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

    Q_INVOKABLE void setChat(const QString &jid);
    Q_INVOKABLE void sendText(const QString &text);
    Q_INVOKABLE void sendFile(const QUrl &fileUrl, const QString &caption);
    Q_INVOKABLE void deleteMessage(const QString &id);
    Q_INVOKABLE void react(const QString &messageId, const QString &emoji);
    Q_INVOKABLE void setReplyTo(const QString &messageId);
    Q_INVOKABLE void clearReply();
    Q_INVOKABLE QString messageIdAt(int index) const;
    Q_INVOKABLE int indexOfMessage(const QString &id) const;
    // Open a message's attachment, downloading it on demand when not yet cached.
    Q_INVOKABLE void openMedia(const QString &id);
    Q_INVOKABLE void editMessage(const QString &id, const QString &text);
    Q_INVOKABLE void toggleStar(const QString &id);
    // Find the next message (from fromRow, in the given direction, wrapping)
    // whose text matches query. Returns its row or -1.
    Q_INVOKABLE int searchFrom(const QString &query, int fromRow, bool forward) const;

    QString replyText() const { return m_replyText; }
    QString replySenderName() const { return m_replySenderName; }
    bool hasReply() const { return !m_replyId.isEmpty(); }

Q_SIGNALS:
    void replyChanged();
    // Emitted when a chat's messages have been loaded and the model rebuilt, so
    // the view can restore its scroll position for the new chat.
    void chatLoaded();
    // Ask the GUI to open a ready local attachment with the system handler.
    void openFileRequested(const QString &path);

private Q_SLOTS:
    void onMessagesReceived(const QJsonArray &messages);
    void onMessageReceived(const QJsonObject &message);
    void onMessageStatus(const QString &chatJid, const QStringList &ids, const QString &status);
    void onMessageMedia(const QString &chatJid, const QString &id, const QString &mediaPath);
    void onMessageEdited(const QString &chatJid, const QString &id, const QString &text);
    void onMessageRevoked(const QString &chatJid, const QString &id);
    void onReaction(const QString &chatJid, const QString &messageId, const QString &senderJid, const QString &emoji, bool fromMe);

private:
    void append(const MessageItem &item);
    void prepend(const MessageItem &item);
    bool renderable(const MessageItem &item) const;
    MessageItem fromJson(const QJsonObject &o) const;
    QString dayLabel(qint64 timestamp) const;

    IpcClient *m_ipc;
    QString m_chatJid;
    QSet<QString> m_pendingOpen; // message ids to open once their download lands
    QList<MessageItem> m_messages;
    QString m_replyId;
    QString m_replySender;
    QString m_replyText;
    QString m_replySenderName;
};
