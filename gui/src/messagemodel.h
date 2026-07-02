#pragma once

#include <QAbstractListModel>
#include <QJsonArray>
#include <QJsonObject>
#include <QList>
#include <QString>

class IpcClient;

struct MessageItem {
    QString id;
    QString senderJid;
    bool fromMe = false;
    qint64 timestamp = 0;
    QString text;
};

// MessageModel holds the messages of the currently open chat. History is loaded
// from the daemon list_messages response; new messages arrive as events.
class MessageModel : public QAbstractListModel
{
    Q_OBJECT

public:
    enum Roles {
        IdRole = Qt::UserRole + 1,
        SenderRole,
        FromMeRole,
        TimestampRole,
        TextRole,
        DayRole,
    };

    explicit MessageModel(IpcClient *ipc, QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = {}) const override;
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

    Q_INVOKABLE void setChat(const QString &jid);
    Q_INVOKABLE void sendText(const QString &text);

private Q_SLOTS:
    void onMessagesReceived(const QJsonArray &messages);
    void onMessageReceived(const QJsonObject &message);

private:
    void append(const MessageItem &item);
    MessageItem fromJson(const QJsonObject &o) const;
    QString dayLabel(qint64 timestamp) const;

    IpcClient *m_ipc;
    QString m_chatJid;
    QList<MessageItem> m_messages;
};
