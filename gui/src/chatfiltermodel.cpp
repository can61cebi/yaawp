#include "chatfiltermodel.h"
#include "chatlistmodel.h"

ChatFilterModel::ChatFilterModel(QObject *parent)
    : QSortFilterProxyModel(parent)
{
    setDynamicSortFilter(true);
}

void ChatFilterModel::setFilterText(const QString &text)
{
    if (m_filterText == text) {
        return;
    }
    m_filterText = text;
    invalidate();
    Q_EMIT filterTextChanged();
}

bool ChatFilterModel::filterAcceptsRow(int sourceRow, const QModelIndex &sourceParent) const
{
    if (m_filterText.isEmpty() || sourceModel() == nullptr) {
        return true;
    }
    const QModelIndex index = sourceModel()->index(sourceRow, 0, sourceParent);
    const QString name = sourceModel()->data(index, ChatListModel::NameRole).toString();
    const QString jid = sourceModel()->data(index, ChatListModel::JidRole).toString();
    return name.contains(m_filterText, Qt::CaseInsensitive)
        || jid.contains(m_filterText, Qt::CaseInsensitive);
}
