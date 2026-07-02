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

find_package(Qt6 6.6 REQUIRED COMPONENTS Core Gui Widgets Qml Quick QuickControls2 Network)
find_package(KF6 6.0 REQUIRED COMPONENTS Kirigami CoreAddons I18n Notifications)

qt_standard_project_setup(REQUIRES 6.6)

qt_add_executable(yaawp
    src/main.cpp
    src/ipcclient.cpp
    src/controller.cpp
    src/chatlistmodel.cpp
    src/messagemodel.cpp
    src/notifier.cpp
)

qt_add_qml_module(yaawp
    URI tr.cebi.yaawp
    VERSION 1.0
    QML_FILES
        qml/Main.qml
        qml/LoginPage.qml
        qml/ChatListPage.qml
        qml/ConversationPage.qml
)

target_link_libraries(yaawp PRIVATE
    Qt6::Core
    Qt6::Gui
    Qt6::Widgets
    Qt6::Qml
    Qt6::Quick
    Qt6::QuickControls2
    Qt6::Network
    KF6::CoreAddons
    KF6::I18n
    KF6::Notifications
)

install(TARGETS yaawp ${KDE_INSTALL_TARGETS_DEFAULT_ARGS})
install(FILES tr.cebi.yaawp.desktop DESTINATION ${KDE_INSTALL_APPDIR})
install(FILES yaawp.notifyrc DESTINATION ${KDE_INSTALL_DATADIR}/knotifications6)
