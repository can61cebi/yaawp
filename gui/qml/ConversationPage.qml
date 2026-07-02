import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import QtQuick.Dialogs
import org.kde.kirigami as Kirigami

Kirigami.Page {
    id: page

    property string chatTitle: "Conversation"
    property string chatJid: ""
    property bool typingActive: false
    property bool restored: false
    readonly property bool isGroup: page.chatJid.endsWith("@g.us")

    title: chatTitle
    padding: 0

    function sendCurrent() {
        const value = input.text.trim()
        if (value.length === 0) {
            return
        }
        MessageModel.sendText(value)
        input.clear()
        page.stopTyping()
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
        const idx = messages.indexAt(Kirigami.Units.largeSpacing, messages.contentY + Kirigami.Units.largeSpacing)
        Controller.saveScroll(page.chatJid, idx)
    }

    function positionInitially() {
        if (Settings.rememberScroll) {
            const idx = Controller.savedScroll(page.chatJid)
            if (idx >= 0 && idx < messages.count) {
                messages.positionViewAtIndex(idx, ListView.Beginning)
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

    FileDialog {
        id: fileDialog
        title: "Send a file"
        onAccepted: {
            MessageModel.sendFile(selectedFile, input.text)
            input.clear()
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
                QQC2.ToolButton {
                    icon.name: "mail-attachment-symbolic"
                    onClicked: fileDialog.open()
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

        onCountChanged: {
            if (!page.restored && count > 0) {
                page.restored = true
                Qt.callLater(page.positionInitially)
            }
        }

        delegate: Item {
            id: row

            required property bool fromMe
            required property string messageId
            required property string text
            required property string type
            required property string mediaPath
            required property var timestamp
            required property string status
            required property string senderName
            required property string reactions
            required property string quotedText
            required property string daySeparator

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
                                visible: bubble.hasMedia
                                source: bubble.hasMedia ? ("file://" + row.mediaPath) : ""
                                fillMode: Image.PreserveAspectFit
                                asynchronous: true
                                sourceSize.width: maxW
                                width: (implicitWidth > 0) ? Math.min(implicitWidth, maxW) : maxW
                                height: (implicitWidth > 0) ? width * (implicitHeight / implicitWidth) : 0
                            }

                            QQC2.Label {
                                id: textLabel
                                readonly property bool revoked: row.type === "revoked"
                                visible: row.text.length > 0 || revoked
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

    QQC2.RoundButton {
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        anchors.rightMargin: Kirigami.Units.largeSpacing + Kirigami.Units.gridUnit
        anchors.bottomMargin: Kirigami.Units.largeSpacing
        focusPolicy: Qt.NoFocus
        // Distance from the newest message. Show only after scrolling up about
        // half a screen so it never covers the latest messages.
        visible: messages.contentHeight > messages.height
                 && (messages.contentHeight - messages.contentY - messages.height) > messages.height * 0.5
        icon.name: "go-down-symbolic"
        onClicked: messages.positionViewAtBeginning()
    }
}
