import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import QtQuick.Dialogs
import org.kde.kirigami as Kirigami
import org.kde.kirigamiaddons.components as KirigamiComponents

Kirigami.Page {
    id: page

    property string chatTitle: "Conversation"
    property string chatJid: ""
    property bool typingActive: false
    property string editingId: ""
    property var forwardData: null
    property bool searchActive: false
    property string searchQuery: ""
    property int searchRow: -1
    readonly property bool isGroup: page.chatJid.endsWith("@g.us")

    title: chatTitle
    padding: 0

    actions: [
        Kirigami.Action {
            text: "Search"
            icon.name: "search"
            onTriggered: {
                page.searchActive = true
                searchField.forceActiveFocus()
            }
        },
        Kirigami.Action {
            text: "Group info"
            icon.name: "documentinfo"
            visible: page.isGroup
            onTriggered: {
                Controller.requestGroupInfo(page.chatJid)
                groupSheet.open()
            }
        },
        Kirigami.Action {
            text: "Contact info"
            icon.name: "documentinfo"
            visible: !page.isGroup
            onTriggered: {
                Controller.requestContactInfo(page.chatJid)
                contactSheet.open()
            }
        }
    ]

    function sendCurrent() {
        const value = input.text.trim()
        if (value.length === 0) {
            return
        }
        if (page.editingId.length > 0) {
            MessageModel.editMessage(page.editingId, value)
            page.cancelEdit()
            page.stopTyping()
            return
        }
        MessageModel.sendText(value)
        input.clear()
        page.stopTyping()
    }

    function startEdit(id, text) {
        MessageModel.clearReply()
        page.editingId = id
        input.text = text
        input.forceActiveFocus()
        input.cursorPosition = input.text.length
    }

    function cancelEdit() {
        page.editingId = ""
        input.clear()
    }

    function startForward(type, text, mediaPath) {
        page.forwardData = { "type": type, "text": text, "mediaPath": mediaPath }
        Ipc.requestChats()
        forwardSheet.open()
    }

    function doForward(jid) {
        if (page.forwardData === null || jid.length === 0) {
            return
        }
        if (page.forwardData.mediaPath.length > 0) {
            Ipc.sendMedia(jid, page.forwardData.mediaPath, "")
        } else if (page.forwardData.type === "text") {
            Ipc.sendText(jid, page.forwardData.text)
        }
        page.forwardData = null
        forwardSheet.close()
    }

    function gotoNextMatch(forward) {
        if (page.searchQuery.length === 0) {
            return
        }
        const r = MessageModel.searchFrom(page.searchQuery, page.searchRow, forward)
        if (r >= 0) {
            page.searchRow = r
            messages.positionViewAtIndex(r, ListView.Center)
        }
    }

    function closeSearch() {
        page.searchActive = false
        page.searchQuery = ""
        page.searchRow = -1
        searchField.text = ""
    }

    function insertEmoji(e) {
        input.insert(input.cursorPosition, e)
        input.forceActiveFocus()
    }

    function startTyping() {
        if (!page.typingActive) {
            Ipc.setTyping(page.chatJid, true)
            page.typingActive = true
        }
        typingTimer.restart()
    }

    function stopTyping() {
        typingTimer.stop()
        if (page.typingActive) {
            Ipc.setTyping(page.chatJid, false)
            page.typingActive = false
        }
    }

    function senderColor(key) {
        var hash = 0
        for (var i = 0; i < key.length; i++) {
            hash = (hash * 31 + key.charCodeAt(i)) % 360
        }
        return Qt.hsla(hash / 360, 0.55, 0.6, 1)
    }

    // Scroll memory is index based: it survives the image loads that change
    // pixel heights, which is why a contentY based approach failed.
    function saveScrollNow() {
        if (!Settings.rememberScroll || page.chatJid.length === 0) {
            return
        }
        Controller.saveScroll(page.chatJid, messages.contentY)
    }

    function positionInitially() {
        // Force a synchronous layout so contentHeight is final (image heights are
        // reserved), then restore the exact offset without any settle jitter.
        messages.forceLayout()
        if (Settings.rememberScroll) {
            const y = Controller.savedScroll(page.chatJid)
            if (y >= 0) {
                messages.contentY = y
                messages.returnToBounds()
            }
        }
        // Otherwise the bottom-up view already shows the newest message.
    }

    Component.onDestruction: {
        page.stopTyping()
        // Only save on a plain back or close; a chat switch saves explicitly
        // before the model changes.
        if (page.chatJid.length > 0 && page.chatJid === Controller.currentChatJid) {
            page.saveScrollNow()
        }
    }

    Timer {
        id: typingTimer
        interval: 4000
        onTriggered: page.stopTyping()
    }

    // Restore the scroll position each time a chat's messages finish loading.
    // This fires even when the new chat has the same message count as the old.
    Connections {
        target: MessageModel
        function onChatLoaded() {
            Qt.callLater(page.positionInitially)
        }
        function onOpenFileRequested(path) {
            Controller.openFile(path)
        }
    }

    FileDialog {
        id: fileDialog
        title: "Send a file"
        onAccepted: {
            MessageModel.sendFile(selectedFile, input.text)
            input.clear()
        }
    }

    Kirigami.OverlaySheet {
        id: groupSheet
        title: (Controller.groupInfo.name !== undefined && Controller.groupInfo.name.length > 0)
               ? Controller.groupInfo.name : page.chatTitle

        ListView {
            implicitWidth: Kirigami.Units.gridUnit * 22
            model: Controller.groupInfo.participants !== undefined ? Controller.groupInfo.participants : []

            header: ColumnLayout {
                width: ListView.view.width
                spacing: Kirigami.Units.smallSpacing

                QQC2.Label {
                    Layout.fillWidth: true
                    visible: text.length > 0
                    text: Controller.groupInfo.topic !== undefined ? Controller.groupInfo.topic : ""
                    wrapMode: Text.WordWrap
                    opacity: 0.8
                }
                QQC2.Label {
                    Layout.fillWidth: true
                    text: (Controller.groupInfo.participant_count !== undefined ? Controller.groupInfo.participant_count : 0) + " participants"
                    font.bold: true
                }
                Kirigami.Separator { Layout.fillWidth: true }
            }

            delegate: QQC2.ItemDelegate {
                width: ListView.view.width
                required property var modelData

                contentItem: RowLayout {
                    spacing: Kirigami.Units.largeSpacing
                    KirigamiComponents.Avatar {
                        Layout.preferredWidth: Kirigami.Units.iconSizes.medium
                        Layout.preferredHeight: Kirigami.Units.iconSizes.medium
                        name: modelData.name
                    }
                    QQC2.Label {
                        Layout.fillWidth: true
                        text: modelData.name
                        elide: Text.ElideRight
                    }
                    QQC2.Label {
                        visible: modelData.is_admin === true
                        text: "admin"
                        opacity: 0.6
                        font.pointSize: Kirigami.Theme.smallFont.pointSize
                    }
                }
            }
        }
    }

    Kirigami.OverlaySheet {
        id: contactSheet
        title: "Contact info"

        ColumnLayout {
            spacing: Kirigami.Units.largeSpacing
            width: Kirigami.Units.gridUnit * 18

            KirigamiComponents.Avatar {
                Layout.alignment: Qt.AlignHCenter
                Layout.preferredWidth: Kirigami.Units.gridUnit * 5
                Layout.preferredHeight: Kirigami.Units.gridUnit * 5
                name: Controller.contactInfo.name !== undefined ? Controller.contactInfo.name : page.chatTitle
                source: (Controller.contactInfo.avatar !== undefined && Controller.contactInfo.avatar.length > 0)
                        ? "file://" + Controller.contactInfo.avatar : ""
            }
            Kirigami.Heading {
                Layout.fillWidth: true
                horizontalAlignment: Text.AlignHCenter
                text: Controller.contactInfo.name !== undefined ? Controller.contactInfo.name : page.chatTitle
                elide: Text.ElideRight
            }
            QQC2.Label {
                Layout.fillWidth: true
                horizontalAlignment: Text.AlignHCenter
                visible: text.length > 0
                text: Controller.contactInfo.phone !== undefined ? Controller.contactInfo.phone : ""
                opacity: 0.8
            }
            QQC2.Label {
                Layout.fillWidth: true
                visible: text.length > 0
                text: Controller.contactInfo.status !== undefined ? Controller.contactInfo.status : ""
                wrapMode: Text.WordWrap
                opacity: 0.8
            }
        }
    }

    Kirigami.OverlaySheet {
        id: forwardSheet
        title: "Forward to"

        ListView {
            implicitWidth: Kirigami.Units.gridUnit * 20
            model: ChatModel

            delegate: QQC2.ItemDelegate {
                id: fwdItem
                width: ListView.view.width
                required property string jid
                required property string name

                contentItem: RowLayout {
                    spacing: Kirigami.Units.largeSpacing
                    KirigamiComponents.Avatar {
                        Layout.preferredWidth: Kirigami.Units.iconSizes.medium
                        Layout.preferredHeight: Kirigami.Units.iconSizes.medium
                        name: fwdItem.name
                    }
                    QQC2.Label {
                        Layout.fillWidth: true
                        text: fwdItem.name
                        elide: Text.ElideRight
                    }
                }

                onClicked: page.doForward(fwdItem.jid)
            }
        }
    }

    header: QQC2.Control {
        visible: Controller.currentChatStatus.length > 0
        height: visible ? implicitHeight : 0
        leftPadding: Kirigami.Units.largeSpacing
        rightPadding: Kirigami.Units.largeSpacing
        topPadding: Kirigami.Units.smallSpacing
        bottomPadding: Kirigami.Units.smallSpacing

        contentItem: QQC2.Label {
            text: Controller.currentChatStatus
            opacity: 0.75
            font: Kirigami.Theme.smallFont
        }
    }

    footer: QQC2.ToolBar {
        contentItem: ColumnLayout {
            spacing: Kirigami.Units.smallSpacing

            RowLayout {
                Layout.fillWidth: true
                visible: MessageModel.hasReply
                spacing: Kirigami.Units.smallSpacing

                Rectangle {
                    Layout.preferredWidth: 3
                    Layout.preferredHeight: replyInfo.implicitHeight
                    radius: 1
                    color: Kirigami.Theme.highlightColor
                }
                ColumnLayout {
                    id: replyInfo
                    Layout.fillWidth: true
                    spacing: 0
                    QQC2.Label {
                        Layout.fillWidth: true
                        visible: text.length > 0
                        text: MessageModel.replySenderName
                        font.bold: true
                        font.pointSize: Kirigami.Theme.smallFont.pointSize
                        elide: Text.ElideRight
                        color: Kirigami.Theme.highlightColor
                    }
                    QQC2.Label {
                        Layout.fillWidth: true
                        text: MessageModel.replyText
                        opacity: 0.8
                        elide: Text.ElideRight
                    }
                }
                QQC2.ToolButton {
                    icon.name: "dialog-close"
                    onClicked: MessageModel.clearReply()
                }
            }

            RowLayout {
                Layout.fillWidth: true
                visible: page.editingId.length > 0
                spacing: Kirigami.Units.smallSpacing

                Kirigami.Icon {
                    source: "document-edit"
                    Layout.preferredWidth: Kirigami.Units.iconSizes.small
                    Layout.preferredHeight: Kirigami.Units.iconSizes.small
                }
                QQC2.Label {
                    Layout.fillWidth: true
                    text: "Editing message"
                    color: Kirigami.Theme.highlightColor
                    elide: Text.ElideRight
                }
                QQC2.ToolButton {
                    icon.name: "dialog-close"
                    onClicked: page.cancelEdit()
                }
            }

            RowLayout {
                Layout.fillWidth: true
                QQC2.ToolButton {
                    icon.name: "mail-attachment-symbolic"
                    onClicked: fileDialog.open()
                }
                QQC2.ToolButton {
                    id: emojiButton
                    icon.name: "smiley"
                    onClicked: emojiPopup.open()

                    QQC2.Popup {
                        id: emojiPopup
                        y: -height - Kirigami.Units.smallSpacing
                        width: Kirigami.Units.gridUnit * 18
                        height: Kirigami.Units.gridUnit * 12
                        padding: Kirigami.Units.smallSpacing

                        readonly property var emojis: [
                            "😀","😁","😂","🤣","😊","😇","🙂","🙃","😉","😌",
                            "😍","🥰","😘","😗","😋","😛","😝","🤪","🤔","🤗",
                            "🤓","😎","🥳","😏","😒","😔","😞","😕","🙁","😣",
                            "😫","😩","🥺","😢","😭","😤","😠","😡","🤯","😳",
                            "🥵","🥶","😱","😨","😰","😅","😓","🤭","🤫","😶",
                            "👍","👎","👌","✌️","🤞","🤙","👏","🙏","💪","🫶",
                            "❤️","🧡","💛","💚","💙","💜","🖤","🔥","🎉","💯"
                        ]

                        contentItem: GridView {
                            clip: true
                            cellWidth: Kirigami.Units.gridUnit * 2
                            cellHeight: Kirigami.Units.gridUnit * 2
                            model: emojiPopup.emojis

                            delegate: QQC2.AbstractButton {
                                id: emojiCell
                                required property var modelData
                                width: GridView.view.cellWidth
                                height: GridView.view.cellHeight
                                hoverEnabled: true

                                background: Rectangle {
                                    radius: Kirigami.Units.smallSpacing
                                    color: emojiCell.hovered ? Kirigami.Theme.highlightColor : "transparent"
                                }
                                contentItem: QQC2.Label {
                                    text: emojiCell.modelData
                                    font.pointSize: Kirigami.Theme.defaultFont.pointSize * 1.4
                                    horizontalAlignment: Text.AlignHCenter
                                    verticalAlignment: Text.AlignVCenter
                                }
                                onClicked: {
                                    page.insertEmoji(emojiCell.modelData)
                                    emojiPopup.close()
                                }
                            }
                        }
                    }
                }
                QQC2.TextField {
                    id: input
                    Layout.fillWidth: true
                    placeholderText: "Type a message"
                    onTextEdited: {
                        if (text.length > 0) {
                            page.startTyping()
                        } else {
                            page.stopTyping()
                        }
                    }
                    onAccepted: page.sendCurrent()
                    Keys.onPressed: (event) => {
                        if ((event.modifiers & Qt.ControlModifier) && event.key === Qt.Key_V) {
                            const p = Controller.takeClipboardImage()
                            if (p.length > 0) {
                                Ipc.sendMedia(page.chatJid, p, "")
                                event.accepted = true
                            }
                        }
                    }
                }
                QQC2.Button {
                    text: "Send"
                    icon.name: "document-send"
                    enabled: input.text.trim().length > 0
                    onClicked: page.sendCurrent()
                }
            }
        }
    }

    ListView {
        id: messages
        anchors.fill: parent
        clip: true
        model: MessageModel
        spacing: Kirigami.Units.smallSpacing
        topMargin: Kirigami.Units.smallSpacing
        bottomMargin: Kirigami.Units.largeSpacing
        cacheBuffer: 800
        // Newest at the bottom, anchored there automatically. This is the
        // pattern KDE NeoChat uses for its timeline.
        verticalLayoutDirection: ListView.BottomToTop
        boundsBehavior: Flickable.StopAtBounds

        QQC2.ScrollBar.vertical: QQC2.ScrollBar {}

        delegate: Item {
            id: row

            required property bool fromMe
            required property string messageId
            required property string text
            required property string type
            required property string mediaPath
            required property int mediaWidth
            required property int mediaHeight
            required property var timestamp
            required property string status
            required property string senderName
            required property string reactions
            required property string quotedText
            required property string daySeparator
            required property bool edited
            required property bool starred

            width: messages.width
            implicitHeight: dcol.height

            Column {
                id: dcol
                width: parent.width
                spacing: 0

                Item {
                    width: parent.width
                    visible: row.daySeparator.length > 0
                    height: visible ? daychip.height + Kirigami.Units.largeSpacing : 0

                    Rectangle {
                        id: daychip
                        anchors.centerIn: parent
                        radius: height / 2
                        color: Kirigami.Theme.alternateBackgroundColor
                        width: daychipLabel.implicitWidth + Kirigami.Units.largeSpacing * 2
                        height: daychipLabel.implicitHeight + Kirigami.Units.smallSpacing

                        QQC2.Label {
                            id: daychipLabel
                            anchors.centerIn: parent
                            text: row.daySeparator
                            font: Kirigami.Theme.smallFont
                            opacity: 0.8
                        }
                    }
                }

                Item {
                    width: parent.width
                    height: bubble.height

                    Rectangle {
                        id: bubble

                        readonly property int hpad: Kirigami.Units.largeSpacing
                        readonly property int vpad: Kirigami.Units.smallSpacing + 2
                        readonly property bool showSender: page.isGroup && !row.fromMe && row.senderName.length > 0
                        readonly property bool hasMedia: (row.type === "image" || row.type === "sticker") && row.mediaPath.length > 0
                        readonly property real maxContent: messages.width * 0.72 - hpad * 2
                        readonly property real contentW: {
                            var w = metaContent.implicitWidth
                            if (textLabel.visible)
                                w = Math.max(w, Math.min(textLabel.implicitWidth, maxContent))
                            if (showSender)
                                w = Math.max(w, Math.min(senderLabel.implicitWidth, maxContent))
                            if (hasMedia)
                                w = Math.max(w, mediaImage.width)
                            if (fileChip.isFile)
                                w = Math.max(w, fileChip.width)
                            if (row.reactions.length > 0)
                                w = Math.max(w, reactionsLabel.implicitWidth)
                            if (row.quotedText.length > 0)
                                w = Math.max(w, quoteLabel.width)
                            return w
                        }

                        anchors.left: row.fromMe ? undefined : parent.left
                        anchors.right: row.fromMe ? parent.right : undefined
                        anchors.leftMargin: Kirigami.Units.largeSpacing
                        anchors.rightMargin: Kirigami.Units.largeSpacing + Kirigami.Units.gridUnit

                        width: contentW + hpad * 2
                        height: content.height + vpad * 2
                        radius: Kirigami.Units.largeSpacing
                        color: row.fromMe ? Kirigami.Theme.highlightColor : Kirigami.Theme.alternateBackgroundColor
                        border.width: (page.searchActive && page.searchQuery.length > 0
                                       && row.text.toLowerCase().indexOf(page.searchQuery.toLowerCase()) >= 0) ? 2 : 0
                        border.color: Kirigami.Theme.neutralTextColor

                        TapHandler {
                            acceptedButtons: Qt.RightButton
                            onTapped: contextMenu.popup()
                        }
                        TapHandler {
                            acceptedDevices: PointerDevice.TouchScreen
                            onLongPressed: contextMenu.popup()
                        }

                        QQC2.Menu {
                            id: contextMenu
                            QQC2.Menu {
                                title: "React"
                                QQC2.MenuItem { text: "\u{1F44D}"; onTriggered: MessageModel.react(row.messageId, text) }
                                QQC2.MenuItem { text: "❤️"; onTriggered: MessageModel.react(row.messageId, text) }
                                QQC2.MenuItem { text: "\u{1F602}"; onTriggered: MessageModel.react(row.messageId, text) }
                                QQC2.MenuItem { text: "\u{1F62E}"; onTriggered: MessageModel.react(row.messageId, text) }
                                QQC2.MenuItem { text: "\u{1F622}"; onTriggered: MessageModel.react(row.messageId, text) }
                                QQC2.MenuItem { text: "\u{1F64F}"; onTriggered: MessageModel.react(row.messageId, text) }
                                QQC2.MenuItem { text: "Remove"; onTriggered: MessageModel.react(row.messageId, "") }
                            }
                            QQC2.MenuItem {
                                text: "Reply"
                                visible: row.type !== "revoked"
                                height: visible ? implicitHeight : 0
                                onTriggered: MessageModel.setReplyTo(row.messageId)
                            }
                            QQC2.MenuItem {
                                text: "Edit"
                                visible: row.fromMe && row.type === "text"
                                height: visible ? implicitHeight : 0
                                onTriggered: page.startEdit(row.messageId, row.text)
                            }
                            QQC2.MenuItem {
                                text: "Forward"
                                visible: row.type !== "revoked" && (row.type === "text" || row.mediaPath.length > 0)
                                height: visible ? implicitHeight : 0
                                onTriggered: page.startForward(row.type, row.text, row.mediaPath)
                            }
                            QQC2.MenuItem {
                                text: row.starred ? "Unstar" : "Star"
                                visible: row.type !== "revoked"
                                height: visible ? implicitHeight : 0
                                onTriggered: MessageModel.toggleStar(row.messageId)
                            }
                            QQC2.MenuItem {
                                text: "Copy"
                                visible: row.text.length > 0 && row.type !== "revoked"
                                height: visible ? implicitHeight : 0
                                onTriggered: Controller.copyToClipboard(row.text)
                            }
                            QQC2.MenuItem {
                                text: "Delete for everyone"
                                visible: row.fromMe && row.type !== "revoked"
                                height: visible ? implicitHeight : 0
                                onTriggered: MessageModel.deleteMessage(row.messageId)
                            }
                        }

                        Column {
                            id: content
                            x: bubble.hpad
                            y: bubble.vpad
                            width: bubble.contentW
                            spacing: Math.round(Kirigami.Units.smallSpacing / 2)

                            QQC2.Label {
                                id: senderLabel
                                visible: bubble.showSender
                                width: parent.width
                                text: row.senderName
                                font.bold: true
                                font.pointSize: Kirigami.Theme.smallFont.pointSize
                                elide: Text.ElideRight
                                color: page.senderColor(row.senderName)
                            }

                            QQC2.Label {
                                id: quoteLabel
                                visible: row.quotedText.length > 0
                                width: Math.min(implicitWidth, bubble.maxContent)
                                leftPadding: Kirigami.Units.smallSpacing
                                text: row.quotedText
                                elide: Text.ElideRight
                                opacity: 0.7
                                font.italic: true
                                font.pointSize: Kirigami.Theme.smallFont.pointSize
                                color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                            }

                            Image {
                                id: mediaImage
                                readonly property real maxW: messages.width * 0.6
                                // Use the dimensions from the daemon so the height
                                // is reserved before the file loads. This keeps the
                                // layout stable and scroll positions accurate.
                                readonly property real natW: row.mediaWidth > 0 ? row.mediaWidth : implicitWidth
                                readonly property real natH: row.mediaHeight > 0 ? row.mediaHeight : implicitHeight
                                visible: bubble.hasMedia
                                source: bubble.hasMedia ? ("file://" + row.mediaPath) : ""
                                fillMode: Image.PreserveAspectFit
                                asynchronous: true
                                sourceSize.width: maxW
                                width: natW > 0 ? Math.min(natW, maxW) : maxW
                                height: natW > 0 ? width * (natH / natW) : maxW * 0.6

                                TapHandler {
                                    acceptedButtons: Qt.LeftButton
                                    onTapped: if (bubble.hasMedia) Controller.openFile(row.mediaPath)
                                }
                            }

                            // Attachment chip for media fetched on demand: video,
                            // audio and documents. Tapping downloads then opens it.
                            Rectangle {
                                id: fileChip
                                readonly property bool isFile: row.type === "video" || row.type === "audio" || row.type === "document"
                                readonly property bool ready: row.mediaPath.length > 0
                                visible: fileChip.isFile
                                width: visible ? fileRow.width + Kirigami.Units.smallSpacing * 3 : 0
                                height: visible ? fileRow.implicitHeight + Kirigami.Units.smallSpacing * 2 : 0
                                radius: Kirigami.Units.smallSpacing
                                color: row.fromMe ? Qt.rgba(0, 0, 0, 0.18) : Qt.rgba(0, 0, 0, 0.10)

                                Row {
                                    id: fileRow
                                    x: Kirigami.Units.smallSpacing
                                    anchors.verticalCenter: parent.verticalCenter
                                    spacing: Kirigami.Units.smallSpacing

                                    Kirigami.Icon {
                                        anchors.verticalCenter: parent.verticalCenter
                                        width: Kirigami.Units.iconSizes.medium
                                        height: Kirigami.Units.iconSizes.medium
                                        source: row.type === "video" ? "media-playback-start"
                                              : row.type === "audio" ? "audio-volume-high"
                                              : "text-x-generic"
                                        color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                                    }
                                    Column {
                                        anchors.verticalCenter: parent.verticalCenter
                                        spacing: 0
                                        QQC2.Label {
                                            width: Math.min(implicitWidth, bubble.maxContent - Kirigami.Units.iconSizes.medium - Kirigami.Units.largeSpacing)
                                            text: row.text.length > 0 ? row.text
                                                : (row.type === "video" ? "Video" : row.type === "audio" ? "Voice message" : "Document")
                                            elide: Text.ElideRight
                                            color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                                        }
                                        QQC2.Label {
                                            text: fileChip.ready ? "Tap to open" : "Tap to download"
                                            font: Kirigami.Theme.smallFont
                                            opacity: 0.7
                                            color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                                        }
                                    }
                                }

                                TapHandler {
                                    onTapped: MessageModel.openMedia(row.messageId)
                                }
                            }

                            QQC2.Label {
                                id: textLabel
                                readonly property bool revoked: row.type === "revoked"
                                visible: (row.text.length > 0 || revoked) && !fileChip.isFile
                                width: parent.width
                                text: revoked ? "This message was deleted" : row.text
                                font.italic: revoked
                                opacity: revoked ? 0.7 : 1.0
                                textFormat: Text.PlainText
                                wrapMode: Text.WrapAtWordBoundaryOrAnywhere
                                color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                            }

                            QQC2.Label {
                                id: reactionsLabel
                                visible: row.reactions.length > 0
                                text: row.reactions
                                font.pointSize: Kirigami.Theme.smallFont.pointSize
                                color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                            }

                            Item {
                                width: parent.width
                                height: metaContent.height

                                Row {
                                    id: metaContent
                                    anchors.right: parent.right
                                    spacing: Kirigami.Units.smallSpacing

                                    Kirigami.Icon {
                                        anchors.verticalCenter: parent.verticalCenter
                                        visible: row.starred
                                        source: "starred-symbolic"
                                        width: Kirigami.Units.iconSizes.small
                                        height: Kirigami.Units.iconSizes.small
                                        opacity: 0.7
                                    }

                                    QQC2.Label {
                                        anchors.verticalCenter: parent.verticalCenter
                                        visible: row.edited && row.type !== "revoked"
                                        text: "edited"
                                        font.pointSize: Kirigami.Theme.smallFont.pointSize
                                        font.italic: true
                                        opacity: 0.6
                                        color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                                    }

                                    QQC2.Label {
                                        anchors.verticalCenter: parent.verticalCenter
                                        text: Qt.formatDateTime(new Date(row.timestamp * 1000), "hh:mm")
                                        font: Kirigami.Theme.smallFont
                                        opacity: 0.7
                                        color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                                    }

                                    Item {
                                        id: ticks
                                        anchors.verticalCenter: parent.verticalCenter
                                        visible: row.fromMe && row.status.length > 0
                                        readonly property real tw: Kirigami.Units.iconSizes.small
                                        readonly property color tc: row.status === "read" ? Kirigami.Theme.positiveTextColor : Kirigami.Theme.highlightedTextColor
                                        readonly property real op: row.status === "read" ? 1.0 : 0.75
                                        width: row.status === "sent" ? tw : tw * 1.5
                                        height: tw

                                        Kirigami.Icon {
                                            source: "checkmark"
                                            width: ticks.tw
                                            height: ticks.tw
                                            x: ticks.width - ticks.tw
                                            color: ticks.tc
                                            opacity: ticks.op
                                        }
                                        Kirigami.Icon {
                                            visible: row.status !== "sent"
                                            source: "checkmark"
                                            width: ticks.tw
                                            height: ticks.tw
                                            x: 0
                                            color: ticks.tc
                                            opacity: ticks.op
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    DropArea {
        id: dropArea
        anchors.fill: parent
        onDropped: (drop) => {
            if (drop.hasUrls) {
                for (let i = 0; i < drop.urls.length; i++) {
                    MessageModel.sendFile(drop.urls[i], "")
                }
                drop.accept()
            }
        }

        Rectangle {
            anchors.fill: parent
            visible: dropArea.containsDrag
            color: Kirigami.Theme.highlightColor
            opacity: 0.15
            border.width: 2
            border.color: Kirigami.Theme.highlightColor

            Kirigami.Heading {
                anchors.centerIn: parent
                text: "Drop to send"
                color: Kirigami.Theme.highlightColor
            }
        }
    }

    QQC2.Pane {
        id: searchBar
        visible: page.searchActive
        z: 10
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.top: parent.top
        padding: Kirigami.Units.smallSpacing

        contentItem: RowLayout {
            spacing: Kirigami.Units.smallSpacing

            QQC2.TextField {
                id: searchField
                Layout.fillWidth: true
                placeholderText: "Search in chat"
                onTextChanged: {
                    page.searchQuery = text
                    page.searchRow = -1
                    page.gotoNextMatch(true)
                }
                onAccepted: page.gotoNextMatch(true)
                Keys.onEscapePressed: page.closeSearch()
            }
            QQC2.ToolButton {
                icon.name: "go-up"
                enabled: page.searchQuery.length > 0
                onClicked: page.gotoNextMatch(true)
            }
            QQC2.ToolButton {
                icon.name: "go-down"
                enabled: page.searchQuery.length > 0
                onClicked: page.gotoNextMatch(false)
            }
            QQC2.ToolButton {
                icon.name: "dialog-close"
                onClicked: page.closeSearch()
            }
        }
    }

    QQC2.RoundButton {
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.rightMargin: Kirigami.Units.largeSpacing + Kirigami.Units.gridUnit
        anchors.bottomMargin: Kirigami.Units.largeSpacing
        focusPolicy: Qt.NoFocus
        // In this bottom-up list originY is the oldest (top) and the newest sits
        // at the maximum contentY (originY + contentHeight - height). Show the
        // button only after scrolling up about half a screen from the newest.
        readonly property real bottomY: messages.originY + messages.contentHeight - messages.height
        visible: messages.contentHeight > messages.height
                 && (bottomY - messages.contentY) > messages.height * 0.5
        icon.name: "go-down-symbolic"
        onClicked: {
            messages.contentY = bottomY
            messages.returnToBounds()
        }
    }
}
