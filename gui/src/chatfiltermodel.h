#pragma once

#include <QSortFilterProxyModel>
#include <QString>

// ChatFilterModel filters the chat list by name or number as the user types.
// It is a proxy over ChatListModel so filtering is incremental and cheap.
class ChatFilterModel : public QSortFilterProxyModel
{
    Q_OBJECT
    Q_PROPERTY(QString filterText READ filterText WRITE setFilterText NOTIFY filterTextChanged)

public:
    explicit ChatFilterModel(QObject *parent = nullptr);

    QString filterText() const { return m_filterText; }
    void setFilterText(const QString &text);

Q_SIGNALS:
    void filterTextChanged();

protected:
    bool filterAcceptsRow(int sourceRow, const QModelIndex &sourceParent) const override;

private:
    QString m_filterText;
};
