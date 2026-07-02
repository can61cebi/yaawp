import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import org.kde.kirigami as Kirigami
import org.kde.kirigamiaddons.components as KirigamiComponents

Kirigami.Page {
    id: page
    title: "Chats"
    padding: 0

    actions: [
        Kirigami.Action {
            text: "Refresh"
            icon.name: "view-refresh"
            onTriggered: Ipc.requestChats()
        },
        Kirigami.Action {
            text: "Settings"
            icon.name: "configure"
            onTriggered: applicationWindow().showSettings()
        }
    ]

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        Kirigami.SearchField {
            id: searchField
            Layout.fillWidth: true
            Layout.margins: Kirigami.Units.smallSpacing
            placeholderText: "Search chats"
            onTextChanged: ChatFilterModel.filterText = text
        }

        Kirigami.InlineMessage {
            Layout.fillWidth: true
            visible: Controller.connectionState !== "connected"
            text: Controller.connectionState === "connecting"
                  ? "Connecting to WhatsApp"
                  : "Disconnected. Trying to reconnect."
            type: Kirigami.MessageType.Information
        }

        ListView {
            id: list
            Layout.fillWidth: true
            Layout.fillHeight: true
            clip: true
            model: ChatFilterModel
            boundsBehavior: Flickable.StopAtBounds

            QQC2.ScrollBar.vertical: QQC2.ScrollBar {}

            delegate: QQC2.ItemDelegate {
                id: item
                // Leave a gutter on the right so the hover highlight does not
                // slide under the scrollbar.
                width: ListView.view.width - Kirigami.Units.gridUnit

                required property string jid
                required property string name
                required property string lastPreview
                required property int unread

                contentItem: RowLayout {
                    spacing: Kirigami.Units.largeSpacing

                    KirigamiComponents.Avatar {
                        Layout.preferredWidth: Kirigami.Units.iconSizes.medium
                        Layout.preferredHeight: Kirigami.Units.iconSizes.medium
                        name: item.name
                    }

                    ColumnLayout {
                        Layout.fillWidth: true
                        spacing: 0
                        QQC2.Label {
                            Layout.fillWidth: true
                            text: item.name
                            font.bold: true
                            elide: Text.ElideRight
                        }
                        QQC2.Label {
                            Layout.fillWidth: true
                            text: item.lastPreview
                            opacity: item.unread > 0 ? 0.9 : 0.7
                            font.bold: item.unread > 0
                            elide: Text.ElideRight
                        }
                    }

                    Rectangle {
                        Layout.alignment: Qt.AlignVCenter
                        visible: item.unread > 0
                        implicitWidth: Math.max(height, badgeLabel.implicitWidth + Kirigami.Units.smallSpacing * 2)
                        implicitHeight: badgeLabel.implicitHeight + Kirigami.Units.smallSpacing
                        radius: height / 2
                        color: Kirigami.Theme.highlightColor

                        QQC2.Label {
                            id: badgeLabel
                            anchors.centerIn: parent
                            text: item.unread > 99 ? "99+" : item.unread
                            color: Kirigami.Theme.highlightedTextColor
                            font.pointSize: Kirigami.Theme.smallFont.pointSize
                            font.bold: true
                        }
                    }
                }

                onClicked: applicationWindow().showConversation(item.jid, item.name)
            }

            Kirigami.PlaceholderMessage {
                anchors.centerIn: parent
                width: parent.width - Kirigami.Units.gridUnit * 4
                visible: list.count === 0
                text: searchField.text.length > 0 ? "No matches" : "No chats yet"
                explanation: searchField.text.length > 0 ? "" : "New messages will appear here once you are connected."
            }
        }
    }
}
