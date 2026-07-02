#include "chatlistmodel.h"
#include "ipcclient.h"

ChatListModel::ChatListModel(IpcClient *ipc, QObject *parent)
    : QAbstractListModel(parent)
    , m_ipc(ipc)
{
    connect(ipc, &IpcClient::chatsReceived, this, &ChatListModel::onChatsReceived);
    connect(ipc, &IpcClient::messageReceived, this, &ChatListModel::onMessageReceived);
}

int ChatListModel::rowCount(const QModelIndex &parent) const
{
    if (parent.isValid()) {
        return 0;
    }
    return static_cast<int>(m_chats.size());
}

QVariant ChatListModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid() || index.row() >= m_chats.size()) {
        return {};
    }
    const ChatItem &c = m_chats.at(index.row());
    switch (role) {
    case JidRole:
        return c.jid;
    case NameRole:
        return c.name.isEmpty() ? c.jid : c.name;
    case LastPreviewRole:
        return c.lastPreview;
    case LastTsRole:
        return c.lastTs;
    case UnreadRole:
        return c.unread;
    default:
        return {};
    }
}

QHash<int, QByteArray> ChatListModel::roleNames() const
{
    return {
        {JidRole, "jid"},
        {NameRole, "name"},
        {LastPreviewRole, "lastPreview"},
        {LastTsRole, "lastTs"},
        {UnreadRole, "unread"},
    };
}

void ChatListModel::onChatsReceived(const QJsonArray &chats)
{
    beginResetModel();
    m_chats.clear();
    for (const QJsonValue &value : chats) {
        const QJsonObject o = value.toObject();
        ChatItem item;
        item.jid = o.value(QStringLiteral("jid")).toString();
        item.name = o.value(QStringLiteral("name")).toString();
        item.lastPreview = o.value(QStringLiteral("last_message_preview")).toString();
        item.lastTs = static_cast<qint64>(o.value(QStringLiteral("last_message_ts")).toDouble());
        item.unread = o.value(QStringLiteral("unread_count")).toInt();
        m_chats.append(item);
    }
    endResetModel();
}

int ChatListModel::indexOfJid(const QString &jid) const
{
    for (int i = 0; i < m_chats.size(); ++i) {
        if (m_chats.at(i).jid == jid) {
            return i;
        }
    }
    return -1;
}

void ChatListModel::upsert(const QString &jid, const QString &preview, qint64 ts)
{
    const int existing = indexOfJid(jid);
    if (existing >= 0) {
        m_chats[existing].lastPreview = preview;
        m_chats[existing].lastTs = ts;
        const QModelIndex idx = index(existing);
        Q_EMIT dataChanged(idx, idx);
        return;
    }
    const int row = static_cast<int>(m_chats.size());
    beginInsertRows({}, row, row);
    ChatItem item;
    item.jid = jid;
    item.lastPreview = preview;
    item.lastTs = ts;
    m_chats.append(item);
    endInsertRows();
}

void ChatListModel::onMessageReceived(const QJsonObject &message)
{
    const QString jid = message.value(QStringLiteral("chat_jid")).toString();
    if (jid.isEmpty()) {
        return;
    }
    upsert(jid,
           message.value(QStringLiteral("text")).toString(),
           static_cast<qint64>(message.value(QStringLiteral("timestamp")).toDouble()));
}
