import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import org.kde.kirigami as Kirigami

Kirigami.ScrollablePage {
    id: page

    property string chatTitle: "Conversation"
    title: chatTitle

    function sendCurrent() {
        const value = input.text.trim()
        if (value.length === 0) {
            return
        }
        MessageModel.sendText(value)
        input.clear()
    }

    footer: QQC2.ToolBar {
        RowLayout {
            anchors.fill: parent
            QQC2.TextField {
                id: input
                Layout.fillWidth: true
                placeholderText: "Type a message"
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
        bottomMargin: Kirigami.Units.smallSpacing
        topMargin: Kirigami.Units.smallSpacing

        delegate: Item {
            id: row

            required property bool fromMe
            required property string text

            width: messages.width
            implicitHeight: bubble.height

            Rectangle {
                id: bubble
                anchors.left: row.fromMe ? undefined : parent.left
                anchors.right: row.fromMe ? parent.right : undefined
                anchors.leftMargin: Kirigami.Units.largeSpacing
                anchors.rightMargin: Kirigami.Units.largeSpacing

                width: label.width
                height: label.height
                radius: Kirigami.Units.largeSpacing
                color: row.fromMe ? Kirigami.Theme.highlightColor : Kirigami.Theme.alternateBackgroundColor

                QQC2.Label {
                    id: label
                    width: Math.min(implicitWidth, messages.width * 0.72)
                    leftPadding: Kirigami.Units.largeSpacing
                    rightPadding: Kirigami.Units.largeSpacing
                    topPadding: Kirigami.Units.smallSpacing * 2
                    bottomPadding: Kirigami.Units.smallSpacing * 2
                    text: row.text
                    textFormat: Text.PlainText
                    wrapMode: Text.WrapAtWordBoundaryOrAnywhere
                    color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                }
            }
        }

        onCountChanged: positionViewAtEnd()
    }
}
