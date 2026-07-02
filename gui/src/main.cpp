#include <QApplication>
#include <QIcon>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQuickStyle>

#include <KLocalizedContext>
#include <KLocalizedString>

#include "chatlistmodel.h"
#include "controller.h"
#include "ipcclient.h"
#include "messagemodel.h"

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

    KLocalizedString::setApplicationDomain("yaawp");

    IpcClient ipc;
    Controller controller(&ipc);
    ChatListModel chatModel(&ipc);
    MessageModel messageModel(&ipc);

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

    ipc.connectToDaemon();
    return app.exec();
}
