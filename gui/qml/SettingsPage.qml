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
}
