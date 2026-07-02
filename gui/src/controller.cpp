#include "controller.h"
#include "ipcclient.h"

Controller::Controller(IpcClient *ipc, QObject *parent)
    : QObject(parent)
    , m_ipc(ipc)
{
    connect(ipc, &IpcClient::qrReceived, this, &Controller::onQrReceived);
    connect(ipc, &IpcClient::connectionStateChanged, this, &Controller::onConnectionStateChanged);
    connect(ipc, &IpcClient::pairSucceeded, this, &Controller::onPairSucceeded);
    connect(ipc, &IpcClient::eventReceived, this, &Controller::onEvent);
}

void Controller::setCurrentChatJid(const QString &jid)
{
    if (m_currentChatJid == jid) {
        return;
    }
    m_currentChatJid = jid;
    Q_EMIT currentChatChanged();
}

void Controller::onQrReceived(const QString &code)
{
    m_qrCode = code;
    Q_EMIT qrCodeChanged();
}

void Controller::onConnectionStateChanged(const QString &state)
{
    m_connectionState = state;
    Q_EMIT connectionStateChanged();
    if (state == QStringLiteral("connected")) {
        m_qrCode.clear();
        Q_EMIT qrCodeChanged();
        m_ipc->requestChats();
    }
}

void Controller::onPairSucceeded(const QString &jid, const QString &pushName)
{
    Q_UNUSED(jid)
    Q_UNUSED(pushName)
    // Pairing is done; a connection event follows and refreshes the UI.
}

void Controller::onEvent(const QString &event, const QJsonObject &data)
{
    Q_UNUSED(data)
    // History sync arrives in batches after linking; refresh the chat list so
    // conversations appear without the user pressing Refresh.
    if (event == QStringLiteral("history_sync")) {
        m_ipc->requestChats();
    }
}
