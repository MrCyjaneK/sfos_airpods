import QtQuick 2.0
import Sailfish.Silica 1.0

CoverBackground {
    id: cover

    property bool connected
    property string deviceName
    property int batteryLeft
    property int batteryRight
    property string noise

    signal setNoise(string mode)

    function level(n) {
        return n < 0 ? "—" : n + "%"
    }

    function toggleAnc() {
        if (!connected)
            return
        setNoise(noise === "anc" ? "off" : "anc")
    }

    Column {
        anchors.centerIn: parent
        width: parent.width - 2 * Theme.paddingLarge
        spacing: Theme.paddingMedium

        Label {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            truncationMode: TruncationMode.Fade
            color: Theme.highlightColor
            text: cover.connected ? (cover.deviceName || "AirPods") : "Not connected"
        }
        Label {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            font.pixelSize: Theme.fontSizeSmall
            color: Theme.secondaryColor
            text: "L " + cover.level(cover.batteryLeft) + "   R " + cover.level(cover.batteryRight)
        }
        BackgroundItem {
            width: parent.width
            height: Theme.itemSizeSmall
            enabled: cover.connected
            onClicked: cover.toggleAnc()

            Label {
                anchors.centerIn: parent
                font.pixelSize: Theme.fontSizeSmall
                color: cover.noise === "anc" ? Theme.highlightColor : Theme.primaryColor
                text: cover.noise === "anc" ? "Canceling" : "Noise cancel"
            }
        }
    }

    CoverActionList {
        CoverAction {
            iconSource: "image://theme/icon-cover-mute"
            onTriggered: cover.toggleAnc()
        }
    }
}
