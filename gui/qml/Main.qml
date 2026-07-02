import QtQuick
import org.kde.kirigami as Kirigami

Kirigami.ApplicationWindow {
    id: root
    title: "yaawp"

    width: 960
    height: 640
    minimumWidth: 500
    minimumHeight: 400

    pageStack.initialPage: Controller.loggedIn ? chatListComponent : loginComponent

    Connections {
        target: Controller
        function onConnectionStateChanged() {
            if (Controller.loggedIn) {
                root.pageStack.replace(chatListComponent)
            } else {
                root.pageStack.replace(loginComponent)
            }
        }
    }

    Component {
        id: loginComponent
        LoginPage {}
    }

    Component {
        id: chatListComponent
        ChatListPage {}
    }
}
