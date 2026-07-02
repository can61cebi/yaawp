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
    connect(ipc, &IpcClient::messageEdited, this, &MessageModel::onMessageEdited);
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
    case MediaWidthRole:
        return m.mediaWidth;
    case MediaHeightRole:
        return m.mediaHeight;
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
    case QuotedIdRole:
        return m.quotedId;
    case EditedRole:
        return m.edited;
    case StarredRole:
        return m.starred;
    case PreviewUrlRole:
        return m.previewUrl;
    case PreviewTitleRole:
        return m.previewTitle;
    case PreviewDescRole:
        return m.previewDesc;
    case PreviewImageRole:
        return m.previewImage;
    case DaySeparatorRole: {
        // Newest first: the older neighbour is at row + 1. Show the day label
        // above the oldest message of each day.
        const int r = index.row();
        if (r + 1 >= m_messages.size()
            || dayLabel(m.timestamp) != dayLabel(m_messages.at(r + 1).timestamp)) {
            return dayLabel(m.timestamp);
        }
        return QString();
    }
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
        {DaySeparatorRole, "daySeparator"},
        {StatusRole, "status"},
        {MediaPathRole, "mediaPath"},
        {MediaWidthRole, "mediaWidth"},
        {MediaHeightRole, "mediaHeight"},
        {ReactionsRole, "reactions"},
        {QuotedTextRole, "quotedText"},
        {QuotedIdRole, "quotedId"},
        {EditedRole, "edited"},
        {StarredRole, "starred"},
        {PreviewUrlRole, "previewUrl"},
        {PreviewTitleRole, "previewTitle"},
        {PreviewDescRole, "previewDesc"},
        {PreviewImageRole, "previewImage"},
    };
}

void MessageModel::setChat(const QString &jid)
{
    m_pendingOpen.clear();
    if (jid.isEmpty()) {
        beginResetModel();
        m_chatJid.clear();
        m_messages.clear();
        endResetModel();
        return;
    }
    // Keep the current messages on screen and just retarget. onMessagesReceived
    // replaces them in a single reset when the history arrives, which avoids an
    // intermediate reset to zero that cancels in-flight delegates and flashes.
    m_chatJid = jid;
    m_ipc->requestMessages(jid);
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
    prepend(item);

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

QString MessageModel::messageIdAt(int index) const
{
    if (index >= 0 && index < m_messages.size()) {
        return m_messages.at(index).id;
    }
    return QString();
}

int MessageModel::indexOfMessage(const QString &id) const
{
    if (id.isEmpty()) {
        return -1;
    }
    for (int i = 0; i < m_messages.size(); ++i) {
        if (m_messages.at(i).id == id) {
            return i;
        }
    }
    return -1;
}

void MessageModel::openMedia(const QString &id)
{
    for (const MessageItem &m : m_messages) {
        if (m.id == id) {
            if (!m.mediaPath.isEmpty()) {
                Q_EMIT openFileRequested(m.mediaPath);
            } else {
                // Fetch on demand; onMessageMedia opens it once it arrives.
                m_pendingOpen.insert(id);
                m_ipc->downloadMedia(m_chatJid, id);
            }
            return;
        }
    }
}

void MessageModel::editMessage(const QString &id, const QString &text)
{
    const QString trimmed = text.trimmed();
    if (trimmed.isEmpty()) {
        return;
    }
    m_ipc->editMessage(m_chatJid, id, trimmed);
}

int MessageModel::searchFrom(const QString &query, int fromRow, bool forward) const
{
    const int n = static_cast<int>(m_messages.size());
    if (query.isEmpty() || n == 0) {
        return -1;
    }
    for (int step = 1; step <= n; ++step) {
        int i = forward ? fromRow + step : fromRow - step;
        i = ((i % n) + n) % n; // wrap into [0, n)
        if (m_messages.at(i).text.contains(query, Qt::CaseInsensitive)) {
            return i;
        }
    }
    return -1;
}

void MessageModel::toggleStar(const QString &id)
{
    for (int i = 0; i < m_messages.size(); ++i) {
        if (m_messages.at(i).id == id) {
            const bool next = !m_messages.at(i).starred;
            m_messages[i].starred = next;
            const QModelIndex idx = index(i);
            Q_EMIT dataChanged(idx, idx, {StarredRole});
            m_ipc->starMessage(m_chatJid, id, next);
            return;
        }
    }
}

void MessageModel::onMessageEdited(const QString &chatJid, const QString &id, const QString &text)
{
    if (chatJid != m_chatJid) {
        return;
    }
    for (int i = 0; i < m_messages.size(); ++i) {
        if (m_messages.at(i).id == id) {
            m_messages[i].text = text;
            m_messages[i].edited = true;
            const QModelIndex idx = index(i);
            Q_EMIT dataChanged(idx, idx, {TextRole, EditedRole});
            return;
        }
    }
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
    item.mediaWidth = o.value(QStringLiteral("media_w")).toInt();
    item.mediaHeight = o.value(QStringLiteral("media_h")).toInt();
    item.edited = o.value(QStringLiteral("edited")).toBool();
    item.starred = o.value(QStringLiteral("starred")).toBool();
    item.quotedId = o.value(QStringLiteral("quoted_id")).toString();
    item.quotedText = o.value(QStringLiteral("quoted_text")).toString();
    item.quotedSender = o.value(QStringLiteral("quoted_sender")).toString();
    item.previewUrl = o.value(QStringLiteral("preview_url")).toString();
    item.previewTitle = o.value(QStringLiteral("preview_title")).toString();
    item.previewDesc = o.value(QStringLiteral("preview_desc")).toString();
    item.previewImage = o.value(QStringLiteral("preview_image")).toString();
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
    QStringList unread;
    // The daemon returns oldest first; store newest first for a bottom-up view.
    for (int i = messages.size() - 1; i >= 0; --i) {
        const MessageItem item = fromJson(messages.at(i).toObject());
        if (!renderable(item)) {
            continue;
        }
        m_messages.append(item);
        if (!item.fromMe && !item.id.isEmpty()) {
            unread.append(item.id);
        }
    }
    endResetModel();
    Q_EMIT chatLoaded();
    m_ipc->markRead(m_chatJid, unread);
}

void MessageModel::append(const MessageItem &item)
{
    const int row = static_cast<int>(m_messages.size());
    beginInsertRows({}, row, row);
    m_messages.append(item);
    endInsertRows();
}

void MessageModel::prepend(const MessageItem &item)
{
    beginInsertRows({}, 0, 0);
    m_messages.prepend(item);
    endInsertRows();
}

bool MessageModel::renderable(const MessageItem &item) const
{
    return !item.text.isEmpty() || !item.mediaPath.isEmpty()
        || item.type == QStringLiteral("revoked");
}

void MessageModel::onMessageReceived(const QJsonObject &message)
{
    if (message.value(QStringLiteral("chat_jid")).toString() != m_chatJid) {
        return;
    }
    MessageItem item = fromJson(message);
    if (!renderable(item)) {
        return;
    }

    // Reconcile with a pending local echo of an outgoing message (near the front).
    if (item.fromMe) {
        for (int i = 0; i < m_messages.size(); ++i) {
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

    prepend(item);

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
            if (m_pendingOpen.remove(id) && !mediaPath.isEmpty()) {
                Q_EMIT openFileRequested(mediaPath);
            }
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
