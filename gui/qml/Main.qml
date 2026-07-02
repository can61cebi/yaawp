import QtQuick
import org.kde.kirigami as Kirigami

Kirigami.ApplicationWindow {
    id: root
    title: "yaawp"

    width: 960
    height: 640
    minimumWidth: 500
    minimumHeight: 400

    property bool showingChats: false

    pageStack.initialPage: Qt.resolvedUrl("LoginPage.qml")

    // Swap the root page only when crossing the logged-in boundary, so the page
    // is not recreated on every connection state change.
    function refreshRoot() {
        if (Controller.loggedIn && !showingChats) {
            pageStack.replace(Qt.resolvedUrl("ChatListPage.qml"))
            showingChats = true
        } else if (!Controller.loggedIn && showingChats) {
            pageStack.replace(Qt.resolvedUrl("LoginPage.qml"))
            showingChats = false
        }
    }

    Connections {
        target: Controller
        function onConnectionStateChanged() {
            root.refreshRoot()
        }
    }

    Component.onCompleted: refreshRoot()
}
