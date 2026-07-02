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

    // Remove any secondary page so the stack is just the chat list again.
    function clearSecondary() {
        while (pageStack.depth > 1) {
            pageStack.pop()
        }
    }

    function showConversation(jid, name) {
        if (currentSecondary === "conversation" && Controller.currentChatJid === jid) {
            return // already open
        }
        if (conversationPage) {
            conversationPage.saveScrollNow()
        }
        Controller.currentChatJid = jid
        if (conversationPage) {
            // Reuse the live page instead of destroying and recreating it on every
            // chat switch. That churn tore the list view down against a resetting
            // model, which is what logged the delegate and scene warnings.
            conversationPage.chatTitle = name
            conversationPage.chatJid = jid
            MessageModel.setChat(jid)
        } else {
            clearSecondary() // drop the settings page if it is open
            MessageModel.setChat(jid)
            conversationPage = pageStack.push(conversationComponent, { chatTitle: name, chatJid: jid })
        }
        currentSecondary = "conversation"
        // Tell the daemon this chat is on screen so its messages are not unread.
        Ipc.setActiveChat(jid)
    }

    function showSettings() {
        if (currentSecondary === "settings") {
            return // already open, do not stack another
        }
        if (conversationPage) {
            conversationPage.saveScrollNow()
            conversationPage = null
        }
        clearSecondary()
        pageStack.push(settingsComponent, {})
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
                Ipc.setActiveChat("")
            }
        }
    }

    // Surface failures of user actions as a passive notification, but stay quiet
    // about background queries that fail routinely while offline.
    Connections {
        target: Ipc
        function onCommandFailed(method, error) {
            const quiet = ["group_info", "contact_info", "list_starred", "request_avatar",
                           "list_messages", "list_chats", "get_state", "set_active_chat",
                           "mark_read", "subscribe_presence"]
            if (quiet.indexOf(method) === -1 && error.length > 0) {
                root.showPassiveNotification("Failed: " + error)
            }
        }
    }

    Component.onCompleted: refreshRoot()

    Component { id: loginComponent; LoginPage {} }
    Component { id: chatListComponent; ChatListPage {} }
    Component { id: conversationComponent; ConversationPage {} }
    Component { id: settingsComponent; SettingsPage {} }
}
