import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import org.kde.kirigami as Kirigami

Kirigami.ScrollablePage {
    id: page

    property string chatTitle: "Conversation"
    property bool typingActive: false
    readonly property bool isGroup: Controller.currentChatJid.endsWith("@g.us")
    title: chatTitle

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
            Ipc.setTyping(Controller.currentChatJid, true)
            page.typingActive = true
        }
        typingTimer.restart()
    }

    function stopTyping() {
        typingTimer.stop()
        if (page.typingActive) {
            Ipc.setTyping(Controller.currentChatJid, false)
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

    Component.onDestruction: page.stopTyping()

    Timer {
        id: typingTimer
        interval: 4000
        onTriggered: page.stopTyping()
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
        RowLayout {
            anchors.fill: parent
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

    ListView {
        id: messages
        model: MessageModel
        spacing: Kirigami.Units.smallSpacing
        topMargin: Kirigami.Units.smallSpacing
        bottomMargin: Kirigami.Units.smallSpacing

        section.property: "day"
        section.criteria: ViewSection.FullString
        section.delegate: Item {
            width: ListView.view.width
            height: chip.height + Kirigami.Units.largeSpacing

            Rectangle {
                id: chip
                anchors.centerIn: parent
                radius: height / 2
                color: Kirigami.Theme.alternateBackgroundColor
                width: chipLabel.implicitWidth + Kirigami.Units.largeSpacing * 2
                height: chipLabel.implicitHeight + Kirigami.Units.smallSpacing

                QQC2.Label {
                    id: chipLabel
                    anchors.centerIn: parent
                    text: section
                    font: Kirigami.Theme.smallFont
                    opacity: 0.8
                }
            }
        }

        delegate: Item {
            id: row

            required property bool fromMe
            required property string text
            required property string type
            required property string mediaPath
            required property var timestamp
            required property string status
            required property string senderName

            width: messages.width
            implicitHeight: bubble.height

            Rectangle {
                id: bubble

                readonly property int hpad: Kirigami.Units.largeSpacing
                readonly property int vpad: Kirigami.Units.smallSpacing + 2
                readonly property bool showSender: page.isGroup && !row.fromMe && row.senderName.length > 0
                readonly property bool hasMedia: (row.type === "image" || row.type === "sticker") && row.mediaPath.length > 0
                readonly property real maxContent: messages.width * 0.72 - hpad * 2
                readonly property real contentW: {
                    var w = metaContent.implicitWidth
                    if (row.text.length > 0)
                        w = Math.max(w, Math.min(textLabel.implicitWidth, maxContent))
                    if (showSender)
                        w = Math.max(w, Math.min(senderLabel.implicitWidth, maxContent))
                    if (hasMedia)
                        w = Math.max(w, mediaImage.width)
                    return w
                }

                anchors.left: row.fromMe ? undefined : parent.left
                anchors.right: row.fromMe ? parent.right : undefined
                anchors.leftMargin: Kirigami.Units.largeSpacing
                anchors.rightMargin: Kirigami.Units.largeSpacing

                width: contentW + hpad * 2
                height: content.height + vpad * 2
                radius: Kirigami.Units.largeSpacing
                color: row.fromMe ? Kirigami.Theme.highlightColor : Kirigami.Theme.alternateBackgroundColor

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
                        visible: row.text.length > 0
                        width: parent.width
                        text: row.text
                        textFormat: Text.PlainText
                        wrapMode: Text.WrapAtWordBoundaryOrAnywhere
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

                            Row {
                                anchors.verticalCenter: parent.verticalCenter
                                visible: row.fromMe && row.status.length > 0
                                spacing: -Kirigami.Units.smallSpacing
                                opacity: row.status === "read" ? 1.0 : 0.55

                                Kirigami.Icon {
                                    source: "checkmark"
                                    width: Kirigami.Units.iconSizes.small
                                    height: width
                                    color: Kirigami.Theme.highlightedTextColor
                                }
                                Kirigami.Icon {
                                    visible: row.status !== "sent"
                                    source: "checkmark"
                                    width: Kirigami.Units.iconSizes.small
                                    height: width
                                    color: Kirigami.Theme.highlightedTextColor
                                }
                            }
                        }
                    }
                }
            }
        }

        onCountChanged: positionViewAtEnd()
    }
}
