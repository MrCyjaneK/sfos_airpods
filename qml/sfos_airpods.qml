import QtQuick 2.0
import Sailfish.Silica 1.0
import Nemo.DBus 2.0
import "pages"
import "cover"

ApplicationWindow {
    id: app

    property bool connected: false
    property string deviceName: ""
    property string deviceAddress: ""
    property int batteryLeft: -1
    property string batteryLeftStatus: ""
    property int batteryRight: -1
    property string batteryRightStatus: ""
    property int batteryCase: -1
    property bool batteryCaseLive: false
    property string batteryCaseStatus: ""
    property string noise: ""
    property string earLeft: ""
    property string earRight: ""
    property string errorText: ""

    function plain(v) {
        if (v && typeof v === "object" && v.value !== undefined)
            return v.value
        return v
    }

    function apply(props) {
        if (!props)
            return
        connected = !!plain(props.Connected)
        deviceName = plain(props.Name) || ""
        deviceAddress = plain(props.Address) || ""
        batteryLeft = Number(plain(props.BatteryLeft))
        batteryLeftStatus = plain(props.BatteryLeftStatus) || ""
        batteryRight = Number(plain(props.BatteryRight))
        batteryRightStatus = plain(props.BatteryRightStatus) || ""
        batteryCase = Number(plain(props.BatteryCase))
        batteryCaseLive = !!plain(props.BatteryCaseLive)
        batteryCaseStatus = plain(props.BatteryCaseStatus) || ""
        noise = plain(props.Noise) || ""
        earLeft = plain(props.EarLeft) || ""
        earRight = plain(props.EarRight) || ""
    }

    function refresh() {
        props.typedCall("GetAll",
            [{ "type": "s", "value": "net.mrcyjanek.sfos_airpods" }],
            function (result) {
                errorText = ""
                if (result && result.length === 1)
                    apply(result[0])
                else
                    apply(result)
            },
            function (err) {
                errorText = "" + err
            })
    }

    function setNoise(mode) {
        errorText = ""
        api.typedCall("SetNoiseMode",
            [{ "type": "s", "value": mode }],
            function () { refresh() },
            function (err) { errorText = "" + err })
    }

    DBusInterface {
        id: props
        bus: DBus.SystemBus
        service: "net.mrcyjanek.sfos_airpods"
        path: "/net/mrcyjanek/sfos_airpods"
        iface: "org.freedesktop.DBus.Properties"
    }

    Timer {
        interval: 200
        running: true
        repeat: true
        onTriggered: refresh()
    }

    DBusInterface {
        id: api
        bus: DBus.SystemBus
        service: "net.mrcyjanek.sfos_airpods"
        path: "/net/mrcyjanek/sfos_airpods"
        iface: "net.mrcyjanek.sfos_airpods"
    }

    Component.onCompleted: refresh()

    cover: CoverPage {
        connected: app.connected
        deviceName: app.deviceName
        batteryLeft: app.batteryLeft
        batteryRight: app.batteryRight
        noise: app.noise
        onSetNoise: app.setNoise(mode)
    }

    initialPage: Component {
        MainPage {
            connected: app.connected
            deviceName: app.deviceName
            deviceAddress: app.deviceAddress
            batteryLeft: app.batteryLeft
            batteryLeftStatus: app.batteryLeftStatus
            batteryRight: app.batteryRight
            batteryRightStatus: app.batteryRightStatus
            batteryCase: app.batteryCase
            batteryCaseLive: app.batteryCaseLive
            batteryCaseStatus: app.batteryCaseStatus
            noise: app.noise
            earLeft: app.earLeft
            earRight: app.earRight
            errorText: app.errorText
            onSetNoise: app.setNoise(mode)
        }
    }
}
