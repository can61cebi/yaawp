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
            onTriggered: applicationWindow().pageStack.push(Qt.resolvedUrl("SettingsPage.qml"))
        }
    ]

    ColumnLayout {
        anchors.fill: parent
        spacing: 0

        Kirigami.InlineMessage {
            Layout.fillWidth: true
            visible: Controller.connectionState !== "connected"
            text: Controller.connectionState === "connecting"
                  ? "Connecting to WhatsApp"
                  : "Disconnected. Trying to reconnect."
            type: Kirigami.MessageType.Information
        }

        QQC2.ScrollView {
            Layout.fillWidth: true
            Layout.fillHeight: true

            ListView {
                id: list
                model: ChatModel

                delegate: QQC2.ItemDelegate {
                    id: item
                    width: ListView.view.width

                    required property string jid
                    required property string name
                    required property string lastPreview

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
                                opacity: 0.7
                                elide: Text.ElideRight
                            }
                        }
                    }

                    onClicked: {
                        MessageModel.setChat(item.jid)
                        Controller.currentChatJid = item.jid
                        applicationWindow().pageStack.push(Qt.resolvedUrl("ConversationPage.qml"), { chatTitle: item.name })
                    }
                }

                Kirigami.PlaceholderMessage {
                    anchors.centerIn: parent
                    width: parent.width - Kirigami.Units.gridUnit * 4
                    visible: list.count === 0
                    text: "No chats yet"
                    explanation: "New messages will appear here once you are connected."
                }
            }
        }
    }
}
