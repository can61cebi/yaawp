import QtQuick
import org.kde.kirigami as Kirigami
import org.kde.kirigamiaddons.formcard as FormCard

FormCard.FormCardPage {
    title: "Settings"

    FormCard.FormHeader {
        title: "Conversations"
    }

    FormCard.FormCard {
        FormCard.FormSwitchDelegate {
            text: "Remember scroll position"
            description: "When you return to a chat, stay where you left off instead of jumping to the newest message."
            checked: Settings.rememberScroll
            onToggled: Settings.rememberScroll = checked
        }
    }

    FormCard.FormHeader {
        title: "Notifications"
    }

    FormCard.FormCard {
        FormCard.FormSwitchDelegate {
            text: "Show notifications"
            description: "Raise a desktop notification for incoming messages that are not from the open chat."
            checked: Settings.notifications
            onToggled: Settings.notifications = checked
        }
    }

    FormCard.FormHeader {
        title: "Appearance"
    }

    FormCard.FormCard {
        FormCard.FormComboBoxDelegate {
            id: scaleCombo
            text: "Message text size"
            model: ["Small", "Normal", "Large", "Larger"]
            readonly property var scales: [0.85, 1.0, 1.15, 1.3]
            currentIndex: {
                const s = Settings.messageScale
                if (s < 0.93) return 0
                if (s < 1.08) return 1
                if (s < 1.22) return 2
                return 3
            }
            onActivated: (index) => Settings.messageScale = scaleCombo.scales[index]
        }
    }
}
