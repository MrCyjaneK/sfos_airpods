import QtQuick 2.0
import Sailfish.Silica 1.0

Item {
    id: row

    property string label
    property string value

    width: parent ? parent.width : 0
    height: Theme.itemSizeExtraSmall

    Label {
        x: Theme.horizontalPageMargin
        anchors.verticalCenter: parent.verticalCenter
        color: Theme.secondaryColor
        text: row.label
    }
    Label {
        anchors.right: parent.right
        anchors.rightMargin: Theme.horizontalPageMargin
        anchors.verticalCenter: parent.verticalCenter
        text: row.value
    }
}
