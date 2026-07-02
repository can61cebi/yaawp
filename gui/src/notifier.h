#pragma once

#include <QJsonArray>
#include <QJsonObject>
#include <QObject>
#include <QSet>
#include <QString>

class IpcClient;
class Controller;

// Notifier raises a native KDE notification for incoming messages that are not
// from the user and do not belong to the chat currently open on screen.
class Notifier : public QObject
{
    Q_OBJECT

public:
    Notifier(IpcClient *ipc, Controller *controller, QObject *parent = nullptr);

private Q_SLOTS:
    void onMessageReceived(const QJsonObject &message);
    void onChatsReceived(const QJsonArray &chats);

private:
    IpcClient *m_ipc;
    Controller *m_controller;
    QSet<QString> m_muted; // jids whose notifications are silenced
};
