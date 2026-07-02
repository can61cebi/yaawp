import QtQuick
import org.kde.kirigami as Kirigami

Kirigami.ApplicationWindow {
    id: root
    title: "yaawp"

    width: 960
    height: 640
    minimumWidth: 500
    minimumHeight: 400

    // Navigation state. The page stack is at most two deep: the chat list plus
    // one secondary page (a conversation or the settings). currentSecondary
    // tracks which secondary page is open so it is never duplicated.
    property bool showingChats: false
    property string currentSecondary: "" // "", "conversation", "settings"
    property var conversationPage: null

    pageStack.initialPage: loginComponent

    // Closing the window hides it to the system tray instead of quitting.
    onClosing: (close) => {
        close.accepted = false
        root.visible = false
    }

    function refreshRoot() {
        if (Controller.loggedIn && !showingChats) {
            pageStack.clear()
            pageStack.push(chatListComponent)
            showingChats = true
            currentSecondary = ""
            conversationPage = null
        } else if (!Controller.loggedIn && showingChats) {
            pageStack.clear()
            pageStack.push(loginComponent)
            showingChats = false
            currentSecondary = ""
            conversationPage = null
        }
    }

    // Show a secondary page, first removing any existing one so the stack stays
    // exactly two deep (chat list plus one secondary page).
    function goSecondary(component, props) {
        while (pageStack.depth > 1) {
            pageStack.pop()
        }
        return pageStack.push(component, props)
    }

    function showConversation(jid, name) {
        if (currentSecondary === "conversation" && Controller.currentChatJid === jid) {
            return // already open
        }
        if (conversationPage) {
            conversationPage.saveScrollNow()
        }
        MessageModel.setChat(jid)
        Controller.currentChatJid = jid
        conversationPage = goSecondary(conversationComponent, { chatTitle: name, chatJid: jid })
        currentSecondary = "conversation"
    }

    function showSettings() {
        if (currentSecondary === "settings") {
            return // already open, do not stack another
        }
        if (conversationPage) {
            conversationPage.saveScrollNow()
            conversationPage = null
        }
        goSecondary(settingsComponent, {})
        currentSecondary = "settings"
    }

    Connections {
        target: Controller
        function onConnectionStateChanged() {
            root.refreshRoot()
        }
    }

    Connections {
        target: root.pageStack
        function onDepthChanged() {
            if (root.pageStack.depth <= 1) {
                root.currentSecondary = ""
                root.conversationPage = null
            }
        }
    }

    Component.onCompleted: refreshRoot()

    Component { id: loginComponent; LoginPage {} }
    Component { id: chatListComponent; ChatListPage {} }
    Component { id: conversationComponent; ConversationPage {} }
    Component { id: settingsComponent; SettingsPage {} }
}
