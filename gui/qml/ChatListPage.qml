import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import org.kde.kirigami as Kirigami

Kirigami.ScrollablePage {
    id: page
    title: "Chats"

    actions: [
        Kirigami.Action {
            text: "Refresh"
            icon.name: "view-refresh"
            onTriggered: Ipc.requestChats()
        }
    ]

    Component {
        id: conversationComponent
        ConversationPage {}
    }

    ListView {
        id: list
        model: ChatModel

        delegate: QQC2.ItemDelegate {
            id: item
            width: ListView.view.width

            required property string jid
            required property string name
            required property string lastPreview

            contentItem: ColumnLayout {
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

            onClicked: {
                MessageModel.setChat(item.jid)
                Controller.currentChatJid = item.jid
                applicationWindow().pageStack.push(conversationComponent, { chatTitle: item.name })
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
