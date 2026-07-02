#pragma once

#include <QAbstractListModel>
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
};

// ChatListModel holds the list of conversations. In this skeleton it is
// populated from incoming message events; later it will be backed by the
// daemon list_chats response and a local store.
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
    };

    explicit ChatListModel(IpcClient *ipc, QObject *parent = nullptr);

    int rowCount(const QModelIndex &parent = {}) const override;
    QVariant data(const QModelIndex &index, int role) const override;
    QHash<int, QByteArray> roleNames() const override;

private Q_SLOTS:
    void onMessageReceived(const QJsonObject &message);

private:
    int indexOfJid(const QString &jid) const;
    void upsert(const QString &jid, const QString &preview, qint64 ts);

    IpcClient *m_ipc;
    QList<ChatItem> m_chats;
};
