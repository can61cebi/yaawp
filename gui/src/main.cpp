#include <QApplication>
#include <QIcon>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQuickStyle>
#include <QWindow>

#include <KLocalizedContext>
#include <KLocalizedString>

#include "chatfiltermodel.h"
#include "chatlistmodel.h"
#include "controller.h"
#include "ipcclient.h"
#include "messagemodel.h"
#include "notifier.h"
#include "settings.h"
#include "tray.h"

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    QApplication::setOrganizationName(QStringLiteral("cebi"));
    QApplication::setOrganizationDomain(QStringLiteral("cebi.tr"));
    QApplication::setApplicationName(QStringLiteral("yaawp"));
    QApplication::setApplicationVersion(QStringLiteral("0.1.0"));
    QApplication::setDesktopFileName(QStringLiteral("tr.cebi.yaawp"));
    // Prefer the themed icon; fall back to the copy compiled into the binary so
    // the window and taskbar have an icon even before the installed hicolor icon
    // is visible to the running session.
    QIcon appIcon = QIcon::fromTheme(QStringLiteral("tr.cebi.yaawp"));
    if (appIcon.isNull()) {
        appIcon = QIcon(QStringLiteral(":/icons/tr.cebi.yaawp.svg"));
    }
    QApplication::setWindowIcon(appIcon);

    // --hidden starts the window in the tray (used by the autostart entry so the
    // app comes up on login and delivers notifications without stealing focus).
    const bool startHidden = app.arguments().contains(QStringLiteral("--hidden"));

    // Use the native KDE desktop style so Qt Quick Controls match Breeze.
    if (QQuickStyle::name().isEmpty()) {
        QQuickStyle::setStyle(QStringLiteral("org.kde.desktop"));
    }

    // Keep running in the system tray when the window is closed.
    QApplication::setQuitOnLastWindowClosed(false);

    KLocalizedString::setApplicationDomain("yaawp");

    IpcClient ipc;
    Controller controller(&ipc);
    controller.setStartHidden(startHidden);
    ChatListModel chatModel(&ipc);
    ChatFilterModel chatFilter;
    chatFilter.setSourceModel(&chatModel);
    MessageModel messageModel(&ipc);
    Settings settings;
    Notifier notifier(&ipc, &controller, &settings);

    QQmlApplicationEngine engine;
    engine.rootContext()->setContextObject(new KLocalizedContext(&engine));
    engine.rootContext()->setContextProperty(QStringLiteral("Ipc"), &ipc);
    engine.rootContext()->setContextProperty(QStringLiteral("Controller"), &controller);
    engine.rootContext()->setContextProperty(QStringLiteral("ChatModel"), &chatModel);
    engine.rootContext()->setContextProperty(QStringLiteral("ChatFilterModel"), &chatFilter);
    engine.rootContext()->setContextProperty(QStringLiteral("MessageModel"), &messageModel);
    engine.rootContext()->setContextProperty(QStringLiteral("Settings"), &settings);

    engine.loadFromModule("tr.cebi.yaawp", "Main");
    if (engine.rootObjects().isEmpty()) {
        return -1;
    }

    Tray tray(&ipc);
    if (auto *window = qobject_cast<QWindow *>(engine.rootObjects().constFirst())) {
        tray.setWindow(window);
    }

    ipc.connectToDaemon();
    return app.exec();
}
