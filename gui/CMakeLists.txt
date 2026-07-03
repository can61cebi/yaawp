cmake_minimum_required(VERSION 3.20)
project(yaawp VERSION 0.1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

find_package(ECM 6.0 REQUIRED NO_MODULE)
set(CMAKE_MODULE_PATH ${ECM_MODULE_PATH})

include(KDEInstallDirs)
include(KDECMakeSettings)
include(ECMQmlModule)

find_package(Qt6 6.6 REQUIRED COMPONENTS Core Gui Widgets Qml Quick QuickControls2 Network Multimedia)
find_package(KF6 6.0 REQUIRED COMPONENTS Kirigami CoreAddons DBusAddons I18n Notifications StatusNotifierItem)

qt_standard_project_setup(REQUIRES 6.6)

qt_add_executable(yaawp
    src/main.cpp
    src/ipcclient.cpp
    src/controller.cpp
    src/chatlistmodel.cpp
    src/chatfiltermodel.cpp
    src/messagemodel.cpp
    src/notifier.cpp
    src/settings.cpp
    src/tray.cpp
)

qt_add_qml_module(yaawp
    URI tr.cebi.yaawp
    VERSION 1.0
    QML_FILES
        qml/Main.qml
        qml/LoginPage.qml
        qml/ChatListPage.qml
        qml/ConversationPage.qml
        qml/SettingsPage.qml
)

# Compile the app icon into the binary so the tray and window icon are always
# available, even before a freshly installed hicolor icon reaches the running
# plasmashell icon cache. Resource path: :/icons/tr.cebi.yaawp.svg
qt_add_resources(yaawp "appicon"
    PREFIX "/"
    FILES icons/tr.cebi.yaawp.svg
)

target_link_libraries(yaawp PRIVATE
    Qt6::Core
    Qt6::Gui
    Qt6::Widgets
    Qt6::Qml
    Qt6::Quick
    Qt6::QuickControls2
    Qt6::Network
    Qt6::Multimedia
    KF6::CoreAddons
    KF6::DBusAddons
    KF6::I18n
    KF6::Notifications
    KF6::StatusNotifierItem
)

install(TARGETS yaawp ${KDE_INSTALL_TARGETS_DEFAULT_ARGS})
install(FILES tr.cebi.yaawp.desktop DESTINATION ${KDE_INSTALL_APPDIR})
install(FILES yaawp.notifyrc DESTINATION ${KDE_INSTALL_DATADIR}/knotifications6)
install(FILES icons/tr.cebi.yaawp.svg
    DESTINATION ${KDE_INSTALL_ICONDIR}/hicolor/scalable/apps)
