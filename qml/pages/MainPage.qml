import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page

    property bool connected
    property string deviceName
    property string deviceAddress
    property int batteryLeft
    property string batteryLeftStatus
    property int batteryRight
    property string batteryRightStatus
    property int batteryCase
    property bool batteryCaseLive
    property string batteryCaseStatus
    property string noise
    property string earLeft
    property string earRight
    property string errorText

    signal setNoise(string mode)

    property bool noiseReady: false

    function budCaption(status) {
        return status === "charging" ? "Charging" : ""
    }

    function caseCaption() {
        if (batteryCase < 0)
            return ""
        if (!batteryCaseLive)
            return "Last known"
        return budCaption(batteryCaseStatus)
    }

    function earSummary() {
        var leftIn = earLeft === "in"
        var rightIn = earRight === "in"
        if (leftIn && rightIn)
            return "Both earbuds"
        if (leftIn)
            return "Left"
        if (rightIn)
            return "Right"
        if (earLeft === "" && earRight === "")
            return "Unknown"
        return "Neither"
    }

    function noiseIndex() {
        var modes = ["off", "anc", "transparency", "adaptive"]
        var i = modes.indexOf(noise)
        return i < 0 ? 0 : i
    }

    onNoiseChanged: {
        if (!noiseBox)
            return
        noiseReady = false
        noiseBox.currentIndex = noiseIndex()
        noiseReady = true
    }

    Component.onCompleted: {
        noiseBox.currentIndex = noiseIndex()
        noiseReady = true
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height + Theme.paddingLarge

        Column {
            id: column
            width: parent.width
            spacing: Theme.paddingLarge

            Item {
                width: parent.width
                height: head.height + Theme.paddingLarge * 2

                Column {
                    id: head
                    anchors.right: parent.right
                    anchors.rightMargin: Theme.horizontalPageMargin
                    anchors.top: parent.top
                    anchors.topMargin: Theme.paddingLarge
                    spacing: Theme.paddingSmall

                    Label {
                        anchors.right: parent.right
                        font.pixelSize: Theme.fontSizeLarge
                        text: page.deviceName || "AirPods"
                    }
                    Label {
                        anchors.right: parent.right
                        color: Theme.secondaryColor
                        font.pixelSize: Theme.fontSizeSmall
                        text: page.connected ? "Connected" : "Not connected"
                    }
                }
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * x
                wrapMode: Text.Wrap
                color: Theme.errorColor
                text: page.errorText
                visible: text !== ""
            }

            Row {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * x

                BatteryMeter {
                    title: "Left"
                    level: page.batteryLeft
                    caption: page.budCaption(page.batteryLeftStatus)
                }
                BatteryMeter {
                    title: "Right"
                    level: page.batteryRight
                    caption: page.budCaption(page.batteryRightStatus)
                }
                BatteryMeter {
                    title: "Case"
                    level: page.batteryCase
                    caption: page.caseCaption()
                }
            }

            InfoRow { label: "In ear"; value: page.earSummary() }

            ComboBox {
                id: noiseBox
                width: parent.width
                label: "Noise control"
                enabled: page.connected
                menu: ContextMenu {
                    MenuItem { text: "Off" }
                    MenuItem { text: "Noise cancellation" }
                    MenuItem { text: "Transparency" }
                    MenuItem { text: "Adaptive" }
                }
                onCurrentIndexChanged: {
                    if (!page.noiseReady)
                        return
                    var modes = ["off", "anc", "transparency", "adaptive"]
                    if (modes[currentIndex] !== page.noise)
                        page.setNoise(modes[currentIndex])
                }
            }
        }

        VerticalScrollDecorator {}
    }
}
