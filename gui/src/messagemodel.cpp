#include "messagemodel.h"
#include "ipcclient.h"

#include <QDateTime>
#include <QLocale>
#include <QStringList>

MessageModel::MessageModel(IpcClient *ipc, QObject *parent)
    : QAbstractListModel(parent)
    , m_ipc(ipc)
{
    connect(ipc, &IpcClient::messagesReceived, this, &MessageModel::onMessagesReceived);
    connect(ipc, &IpcClient::messageReceived, this, &MessageModel::onMessageReceived);
    connect(ipc, &IpcClient::messageStatusChanged, this, &MessageModel::onMessageStatus);
    connect(ipc, &IpcClient::messageMediaChanged, this, &MessageModel::onMessageMedia);
    connect(ipc, &IpcClient::messageRevoked, this, &MessageModel::onMessageRevoked);
    connect(ipc, &IpcClient::reactionReceived, this, &MessageModel::onReaction);
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
    case SenderNameRole:
        return m.senderName;
    case FromMeRole:
        return m.fromMe;
    case TimestampRole:
        return m.timestamp;
    case TextRole:
        return m.text;
    case TypeRole:
        return m.type;
    case DayRole:
        return dayLabel(m.timestamp);
    case StatusRole:
        return m.status;
    case MediaPathRole:
        return m.mediaPath;
    case ReactionsRole: {
        QStringList distinct;
        for (const QString &emoji : m.reactions) {
            if (!emoji.isEmpty() && !distinct.contains(emoji)) {
                distinct.append(emoji);
            }
        }
        return distinct.join(QString());
    }
    case QuotedTextRole:
        return m.quotedText;
    default:
        return {};
    }
}

QHash<int, QByteArray> MessageModel::roleNames() const
{
    return {
        {IdRole, "messageId"},
        {SenderRole, "sender"},
        {SenderNameRole, "senderName"},
        {FromMeRole, "fromMe"},
        {TimestampRole, "timestamp"},
        {TextRole, "text"},
        {TypeRole, "type"},
        {DayRole, "day"},
        {StatusRole, "status"},
        {MediaPathRole, "mediaPath"},
        {ReactionsRole, "reactions"},
        {QuotedTextRole, "quotedText"},
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
    m_ipc->sendText(m_chatJid, text, m_replyId, m_replySender, m_replyText);

    // Local echo for instant feedback. The daemon broadcasts the stored copy,
    // which reconciles this entry (see onMessageReceived).
    MessageItem item;
    item.fromMe = true;
    item.text = text;
    item.status = QStringLiteral("sent");
    item.timestamp = QDateTime::currentSecsSinceEpoch();
    item.quotedText = m_replyText;
    item.quotedSender = m_replySender;
    item.pending = true;
    append(item);

    clearReply();
}

void MessageModel::sendFile(const QUrl &fileUrl, const QString &caption)
{
    const QString path = fileUrl.toLocalFile();
    if (m_chatJid.isEmpty() || path.isEmpty()) {
        return;
    }
    m_ipc->sendMedia(m_chatJid, path, caption);
}

void MessageModel::setReplyTo(const QString &messageId)
{
    for (const MessageItem &m : m_messages) {
        if (m.id == messageId) {
            m_replyId = m.id;
            m_replySender = m.senderJid;
            m_replyText = m.text.isEmpty() ? QStringLiteral("[media]") : m.text;
            m_replySenderName = m.fromMe ? QStringLiteral("You") : m.senderName;
            Q_EMIT replyChanged();
            return;
        }
    }
}

void MessageModel::clearReply()
{
    if (m_replyId.isEmpty()) {
        return;
    }
    m_replyId.clear();
    m_replySender.clear();
    m_replyText.clear();
    m_replySenderName.clear();
    Q_EMIT replyChanged();
}

void MessageModel::deleteMessage(const QString &id)
{
    if (!id.isEmpty()) {
        m_ipc->deleteMessage(m_chatJid, id);
    }
}

void MessageModel::react(const QString &messageId, const QString &emoji)
{
    for (const MessageItem &m : m_messages) {
        if (m.id == messageId) {
            m_ipc->sendReaction(m_chatJid, messageId, m.senderJid, m.fromMe, emoji);
            return;
        }
    }
}

MessageItem MessageModel::fromJson(const QJsonObject &o) const
{
    MessageItem item;
    item.id = o.value(QStringLiteral("id")).toString();
    item.senderJid = o.value(QStringLiteral("sender_jid")).toString();
    item.senderName = o.value(QStringLiteral("sender_name")).toString();
    item.fromMe = o.value(QStringLiteral("from_me")).toBool();
    item.timestamp = static_cast<qint64>(o.value(QStringLiteral("timestamp")).toDouble());
    item.text = o.value(QStringLiteral("text")).toString();
    item.type = o.value(QStringLiteral("type")).toString();
    item.status = o.value(QStringLiteral("status")).toString();
    item.mediaPath = o.value(QStringLiteral("media_path")).toString();
    item.quotedText = o.value(QStringLiteral("quoted_text")).toString();
    item.quotedSender = o.value(QStringLiteral("quoted_sender")).toString();
    const QJsonObject reacts = o.value(QStringLiteral("reactions")).toObject();
    for (auto it = reacts.constBegin(); it != reacts.constEnd(); ++it) {
        item.reactions.insert(it.key(), it.value().toString());
    }
    return item;
}

QString MessageModel::dayLabel(qint64 timestamp) const
{
    const QDate date = QDateTime::fromSecsSinceEpoch(timestamp).date();
    const QDate today = QDate::currentDate();
    if (date == today) {
        return QStringLiteral("Today");
    }
    if (date == today.addDays(-1)) {
        return QStringLiteral("Yesterday");
    }
    return QLocale().toString(date, QLocale::LongFormat);
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
        const MessageItem item = fromJson(value.toObject());
        if (item.text.isEmpty()) {
            continue; // skip messages with no renderable text
        }
        m_messages.append(item);
    }
    endResetModel();

    // Mark the incoming messages in this history batch as read.
    QStringList unread;
    for (const MessageItem &m : m_messages) {
        if (!m.fromMe && !m.id.isEmpty()) {
            unread.append(m.id);
        }
    }
    m_ipc->markRead(m_chatJid, unread);
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
    MessageItem item = fromJson(message);
    if (item.text.isEmpty()) {
        return;
    }

    // Reconcile with a pending local echo of an outgoing message.
    if (item.fromMe) {
        for (int i = m_messages.size() - 1; i >= 0; --i) {
            if (m_messages.at(i).pending && m_messages.at(i).fromMe
                && m_messages.at(i).text == item.text) {
                item.pending = false;
                m_messages[i] = item;
                const QModelIndex idx = index(i);
                Q_EMIT dataChanged(idx, idx);
                return;
            }
        }
    }

    append(item);

    // The chat is open, so mark an incoming message as read right away.
    if (!item.fromMe && !item.id.isEmpty()) {
        m_ipc->markRead(m_chatJid, {item.id});
    }
}

void MessageModel::onMessageStatus(const QString &chatJid, const QStringList &ids, const QString &status)
{
    if (chatJid != m_chatJid) {
        return;
    }
    for (int i = 0; i < m_messages.size(); ++i) {
        if (ids.contains(m_messages.at(i).id)) {
            m_messages[i].status = status;
            const QModelIndex idx = index(i);
            Q_EMIT dataChanged(idx, idx, {StatusRole});
        }
    }
}

void MessageModel::onMessageMedia(const QString &chatJid, const QString &id, const QString &mediaPath)
{
    if (chatJid != m_chatJid) {
        return;
    }
    for (int i = 0; i < m_messages.size(); ++i) {
        if (m_messages.at(i).id == id) {
            m_messages[i].mediaPath = mediaPath;
            const QModelIndex idx = index(i);
            Q_EMIT dataChanged(idx, idx, {MediaPathRole});
            return;
        }
    }
}

void MessageModel::onMessageRevoked(const QString &chatJid, const QString &id)
{
    if (chatJid != m_chatJid) {
        return;
    }
    for (int i = 0; i < m_messages.size(); ++i) {
        if (m_messages.at(i).id == id) {
            m_messages[i].type = QStringLiteral("revoked");
            m_messages[i].text.clear();
            m_messages[i].mediaPath.clear();
            const QModelIndex idx = index(i);
            Q_EMIT dataChanged(idx, idx);
            return;
        }
    }
}

void MessageModel::onReaction(const QString &chatJid, const QString &messageId, const QString &senderJid, const QString &emoji, bool fromMe)
{
    Q_UNUSED(fromMe)
    if (chatJid != m_chatJid) {
        return;
    }
    for (int i = 0; i < m_messages.size(); ++i) {
        if (m_messages.at(i).id == messageId) {
            if (emoji.isEmpty()) {
                m_messages[i].reactions.remove(senderJid);
            } else {
                m_messages[i].reactions.insert(senderJid, emoji);
            }
            const QModelIndex idx = index(i);
            Q_EMIT dataChanged(idx, idx, {ReactionsRole});
            return;
        }
    }
}
