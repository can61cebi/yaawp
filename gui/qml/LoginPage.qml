import QtQuick
import QtQuick.Layouts
import QtQuick.Controls as QQC2
import org.kde.kirigami as Kirigami
import org.kde.prison as Prison

Kirigami.Page {
    id: page
    title: "Sign in"

    ColumnLayout {
        anchors.centerIn: parent
        spacing: Kirigami.Units.largeSpacing

        Kirigami.Heading {
            Layout.alignment: Qt.AlignHCenter
            level: 2
            text: "Link this device"
        }

        QQC2.Label {
            Layout.alignment: Qt.AlignHCenter
            Layout.maximumWidth: Kirigami.Units.gridUnit * 20
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.WordWrap
            text: "Open WhatsApp on your phone, go to Linked Devices, and scan the code below."
        }

        Rectangle {
            Layout.alignment: Qt.AlignHCenter
            Layout.preferredWidth: Kirigami.Units.gridUnit * 16
            Layout.preferredHeight: Kirigami.Units.gridUnit * 16
            visible: Controller.qrCode.length > 0
            color: "white"
            radius: Kirigami.Units.smallSpacing

            Prison.Barcode {
                anchors.fill: parent
                anchors.margins: Kirigami.Units.largeSpacing
                barcodeType: Prison.Barcode.QRCode
                content: Controller.qrCode
            }
        }

        QQC2.BusyIndicator {
            Layout.alignment: Qt.AlignHCenter
            running: Controller.qrCode.length === 0
            visible: running
        }

        QQC2.Label {
            Layout.alignment: Qt.AlignHCenter
            opacity: 0.7
            text: "Connection: " + Controller.connectionState
        }
    }
}
