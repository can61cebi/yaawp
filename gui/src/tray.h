#pragma once

#include <QHash>
#include <QObject>
#include <QString>

class KStatusNotifierItem;
class QWindow;
class IpcClient;

// Tray shows a KDE system tray icon and links it to the main window so the
// application keeps running when the window is closed. It also reflects the
// total unread count in its tooltip and attention state.
class Tray : public QObject
{
    Q_OBJECT

public:
    explicit Tray(IpcClient *ipc, QObject *parent = nullptr);

    void setWindow(QWindow *window);

private Q_SLOTS:
    void onChatUnread(const QString &jid, int unread);

private:
    void refreshTooltip();

    KStatusNotifierItem *m_sni;
    QHash<QString, int> m_unread; // per chat unread, summed for the tooltip
};
