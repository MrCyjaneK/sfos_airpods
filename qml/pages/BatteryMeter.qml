import QtQuick 2.0
import Sailfish.Silica 1.0

Item {
    id: meter

    property string title
    property int level
    property string caption

    width: parent ? parent.width / 3 : 0
    height: column.implicitHeight

    Column {
        id: column
        width: parent.width
        spacing: Theme.paddingSmall

        Label {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            color: Theme.secondaryColor
            font.pixelSize: Theme.fontSizeSmall
            text: meter.title
        }
        Label {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            color: Theme.highlightColor
            font.pixelSize: Theme.fontSizeLarge
            text: meter.level < 0 ? "—" : meter.level + "%"
        }
        Item {
            width: parent.width * 0.7
            height: Math.max(2, Theme.paddingSmall / 2)
            x: (parent.width - width) / 2

            Rectangle {
                anchors.fill: parent
                radius: height / 2
                color: Theme.secondaryHighlightColor
                opacity: 0.45
            }
            Rectangle {
                width: parent.width * Math.max(0, Math.min(meter.level, 100)) / 100
                height: parent.height
                radius: height / 2
                color: Theme.highlightColor
                visible: meter.level >= 0
            }
        }
        Label {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            color: Theme.secondaryColor
            font.pixelSize: Theme.fontSizeExtraSmall
            text: meter.caption
        }
    }
}
