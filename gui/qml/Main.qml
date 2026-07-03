import QtQuick
import org.kde.kirigami as Kirigami

Kirigami.ApplicationWindow {
    id: root
    title: "yaawp"

    width: 960
    height: 640
    minimumWidth: 500
    minimumHeight: 400

    // Launched with --hidden (autostart on login): come up in the tray only.
    // The tray icon restores the window on click.
    visible: !Controller.startHidden

    // Navigation state. The page stack is at most two deep: the chat list plus
    // one secondary page (a conversation or the settings). currentSecondary
    // tracks which secondary page is open so it is never duplicated.
    property bool showingChats: false
    property string currentSecondary: "" // "", "conversation", "settings"
    property var conversationPage: null

    // Start on the sign-in screen only when the device is genuinely unpaired.
    // A paired session that is still connecting opens straight on the chat list.
    pageStack.initialPage: Controller.needsLogin ? loginComponent : chatListComponent

    // Closing the window hides it to the system tray instead of quitting.
    onClosing: (close) => {
        close.accepted = false
        root.visible = false
    }

    function refreshRoot() {
        // The root is the chat list unless the device is unpaired, in which case
        // it is the sign-in screen. Merely connecting or disconnected keeps the
        // chat list up (it shows its own reconnecting banner).
        var wantChats = !Controller.needsLogin
        if (wantChats === showingChats) {
            return
        }
        pageStack.clear()
        pageStack.push(wantChats ? chatListComponent : loginComponent)
        showingChats = wantChats
        currentSecondary = ""
        conversationPage = null
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

    // Push the privacy preferences to the daemon on connect and on change, since
    // the daemon holds them only in memory.
    function pushPrivacy() {
        Ipc.setPrivacy(Settings.readReceipts, Settings.shareOnline)
    }

    Connections {
        target: Controller
        function onConnectionStateChanged() {
            root.refreshRoot()
            if (Controller.loggedIn) {
                root.pushPrivacy()
            }
        }
    }

    Connections {
        target: Settings
        function onReadReceiptsChanged() { root.pushPrivacy() }
        function onShareOnlineChanged() { root.pushPrivacy() }
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

    Component.onCompleted: {
        // Match the tracker to whatever initialPage chose, so the first
        // refreshRoot is a no-op instead of re-pushing the same root.
        showingChats = !Controller.needsLogin
        refreshRoot()
    }

    Component { id: loginComponent; LoginPage {} }
    Component { id: chatListComponent; ChatListPage {} }
    Component { id: conversationComponent; ConversationPage {} }
    Component { id: settingsComponent; SettingsPage {} }
}
