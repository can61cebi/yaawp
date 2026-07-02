#pragma once

#include <QAbstractListModel>
#include <QJsonArray>
#include <QJsonObject>
#include <QList>
#include <QString>

class IpcClient;

struct ChatItem {
    QString jid;
    QString name;
    QString lastPreview;
    qint64 lastTs = 0;
    int unread = 0;
    bool pinned = false;
    bool muted = false;
};

// ChatListModel holds the list of conversations. It is filled from the daemon
// list_chats response and kept current by incoming message events.
class ChatListModel : public QAbstractListModel
{
    Q_OBJECT

public:
    enum Roles {
        JidRole = Qt::UserRole + 1,
        NameRole,
        LastPreviewRole,
        LastTsRole,
        UnreadRole,
        PinnedRole,
        MutedRole,
    };

    explicit ChatListModel(IpcClient *ipc, QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = {}) const override;
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

private Q_SLOTS:
    void onChatsReceived(const QJsonArray &chats);
    void onMessageReceived(const QJsonObject &message);
    void onChatUnread(const QString &jid, int unread);

private:
    int indexOfJid(const QString &jid) const;
    void upsert(const QString &jid, const QString &preview, qint64 ts);

    IpcClient *m_ipc;
    QList<ChatItem> m_chats;
};
