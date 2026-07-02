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

        delegate: Item {
            id: row
            width: messages.width
            height: bubble.height + Kirigami.Units.smallSpacing

            required property bool fromMe
            required property string text

            Rectangle {
                id: bubble
                anchors.right: row.fromMe ? parent.right : undefined
                anchors.left: row.fromMe ? undefined : parent.left
                anchors.margins: Kirigami.Units.largeSpacing
                width: Math.min(label.implicitWidth + Kirigami.Units.largeSpacing * 2, messages.width * 0.75)
                height: label.implicitHeight + Kirigami.Units.largeSpacing
                radius: Kirigami.Units.smallSpacing
                color: row.fromMe ? Kirigami.Theme.highlightColor : Kirigami.Theme.alternateBackgroundColor

                QQC2.Label {
                    id: label
                    anchors.fill: parent
                    anchors.margins: Kirigami.Units.largeSpacing
                    text: row.text
                    wrapMode: Text.Wrap
                    color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                }
            }
        }

        onCountChanged: positionViewAtEnd()
    }
}
