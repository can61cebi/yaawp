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
            required property var timestamp

            width: messages.width
            implicitHeight: bubble.height

            Rectangle {
                id: bubble

                readonly property int hpad: Kirigami.Units.largeSpacing
                readonly property int vpad: Kirigami.Units.smallSpacing + 2
                readonly property real maxContent: messages.width * 0.72 - hpad * 2
                readonly property real contentWidth: Math.max(Math.min(textLabel.implicitWidth, maxContent),
                                                              timeLabel.implicitWidth)

                anchors.left: row.fromMe ? undefined : parent.left
                anchors.right: row.fromMe ? parent.right : undefined
                anchors.leftMargin: Kirigami.Units.largeSpacing
                anchors.rightMargin: Kirigami.Units.largeSpacing

                width: contentWidth + hpad * 2
                height: textLabel.height + timeLabel.height + vpad * 2
                radius: Kirigami.Units.largeSpacing
                color: row.fromMe ? Kirigami.Theme.highlightColor : Kirigami.Theme.alternateBackgroundColor

                QQC2.Label {
                    id: textLabel
                    x: bubble.hpad
                    y: bubble.vpad
                    width: bubble.contentWidth
                    text: row.text
                    textFormat: Text.PlainText
                    wrapMode: Text.WrapAtWordBoundaryOrAnywhere
                    color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                }

                QQC2.Label {
                    id: timeLabel
                    x: bubble.width - width - bubble.hpad
                    y: textLabel.y + textLabel.height
                    text: Qt.formatDateTime(new Date(row.timestamp * 1000), "hh:mm")
                    font: Kirigami.Theme.smallFont
                    opacity: 0.7
                    color: row.fromMe ? Kirigami.Theme.highlightedTextColor : Kirigami.Theme.textColor
                }
            }
        }

        onCountChanged: positionViewAtEnd()
    }
}
