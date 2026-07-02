#include <QApplication>
#include <QIcon>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQuickStyle>
#include <QWindow>

#include <KLocalizedContext>
#include <KLocalizedString>

#include "chatlistmodel.h"
#include "controller.h"
#include "ipcclient.h"
#include "messagemodel.h"
#include "notifier.h"
#include "tray.h"

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    QApplication::setOrganizationName(QStringLiteral("cebi"));
    QApplication::setOrganizationDomain(QStringLiteral("cebi.tr"));
    QApplication::setApplicationName(QStringLiteral("yaawp"));
    QApplication::setDesktopFileName(QStringLiteral("tr.cebi.yaawp"));
    QApplication::setWindowIcon(QIcon::fromTheme(QStringLiteral("internet-mail")));

    // Use the native KDE desktop style so Qt Quick Controls match Breeze.
    if (QQuickStyle::name().isEmpty()) {
        QQuickStyle::setStyle(QStringLiteral("org.kde.desktop"));
    }

    // Keep running in the system tray when the window is closed.
    QApplication::setQuitOnLastWindowClosed(false);

    KLocalizedString::setApplicationDomain("yaawp");

    IpcClient ipc;
    Controller controller(&ipc);
    ChatListModel chatModel(&ipc);
    MessageModel messageModel(&ipc);
    Notifier notifier(&ipc, &controller);

    QQmlApplicationEngine engine;
    engine.rootContext()->setContextObject(new KLocalizedContext(&engine));
    engine.rootContext()->setContextProperty(QStringLiteral("Ipc"), &ipc);
    engine.rootContext()->setContextProperty(QStringLiteral("Controller"), &controller);
    engine.rootContext()->setContextProperty(QStringLiteral("ChatModel"), &chatModel);
    engine.rootContext()->setContextProperty(QStringLiteral("MessageModel"), &messageModel);

    engine.loadFromModule("tr.cebi.yaawp", "Main");
    if (engine.rootObjects().isEmpty()) {
        return -1;
    }

    Tray tray;
    if (auto *window = qobject_cast<QWindow *>(engine.rootObjects().constFirst())) {
        tray.setWindow(window);
    }

    ipc.connectToDaemon();
    return app.exec();
}
