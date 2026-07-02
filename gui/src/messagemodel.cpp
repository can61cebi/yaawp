#include "messagemodel.h"
#include "ipcclient.h"

MessageModel::MessageModel(IpcClient *ipc, QObject *parent)
    : QAbstractListModel(parent)
    , m_ipc(ipc)
{
    connect(ipc, &IpcClient::messagesReceived, this, &MessageModel::onMessagesReceived);
    connect(ipc, &IpcClient::messageReceived, this, &MessageModel::onMessageReceived);
}

int MessageModel::rowCount(const QModelIndex &parent) const
{
    if (parent.isValid()) {
        return 0;
    }
    return static_cast<int>(m_messages.size());
}

QVariant MessageModel::data(const QModelIndex &index, int role) const
{
    if (!index.isValid() || index.row() >= m_messages.size()) {
        return {};
    }
    const MessageItem &m = m_messages.at(index.row());
    switch (role) {
    case IdRole:
        return m.id;
    case SenderRole:
        return m.senderJid;
    case FromMeRole:
        return m.fromMe;
    case TimestampRole:
        return m.timestamp;
    case TextRole:
        return m.text;
    default:
        return {};
    }
}

QHash<int, QByteArray> MessageModel::roleNames() const
{
    return {
        {IdRole, "messageId"},
        {SenderRole, "sender"},
        {FromMeRole, "fromMe"},
        {TimestampRole, "timestamp"},
        {TextRole, "text"},
    };
}

void MessageModel::setChat(const QString &jid)
{
    beginResetModel();
    m_chatJid = jid;
    m_messages.clear();
    endResetModel();
    if (!jid.isEmpty()) {
        m_ipc->requestMessages(jid);
    }
}

void MessageModel::sendText(const QString &text)
{
    if (m_chatJid.isEmpty() || text.isEmpty()) {
        return;
    }
    m_ipc->sendText(m_chatJid, text);

    // Local echo; the server message and receipt reconcile this later.
    MessageItem item;
    item.fromMe = true;
    item.text = text;
    append(item);
}

MessageItem MessageModel::fromJson(const QJsonObject &o) const
{
    MessageItem item;
    item.id = o.value(QStringLiteral("id")).toString();
    item.senderJid = o.value(QStringLiteral("sender_jid")).toString();
    item.fromMe = o.value(QStringLiteral("from_me")).toBool();
    item.timestamp = static_cast<qint64>(o.value(QStringLiteral("timestamp")).toDouble());
    item.text = o.value(QStringLiteral("text")).toString();
    return item;
}

void MessageModel::onMessagesReceived(const QJsonArray &messages)
{
    // Ignore a history batch that does not belong to the open chat.
    if (!messages.isEmpty()) {
        const QString jid = messages.first().toObject().value(QStringLiteral("chat_jid")).toString();
        if (jid != m_chatJid) {
            return;
        }
    }
    beginResetModel();
    m_messages.clear();
    for (const QJsonValue &value : messages) {
        m_messages.append(fromJson(value.toObject()));
    }
    endResetModel();
}

void MessageModel::append(const MessageItem &item)
{
    const int row = static_cast<int>(m_messages.size());
    beginInsertRows({}, row, row);
    m_messages.append(item);
    endInsertRows();
}

void MessageModel::onMessageReceived(const QJsonObject &message)
{
    if (message.value(QStringLiteral("chat_jid")).toString() != m_chatJid) {
        return;
    }
    append(fromJson(message));
}
