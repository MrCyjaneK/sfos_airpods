TARGET = sfos_airpods

CONFIG += sailfishapp c++11

SOURCES += src/sfos_airpods.cpp

DISTFILES += qml/sfos_airpods.qml \
    qml/cover/CoverPage.qml \
    qml/pages/MainPage.qml \
    qml/pages/BatteryMeter.qml \
    qml/pages/InfoRow.qml \
    rpm/sfos_airpods.spec \
    sfos_airpods.desktop \
    icons/86x86/sfos_airpods.png \
    icons/108x108/sfos_airpods.png \
    icons/128x128/sfos_airpods.png \
    icons/172x172/sfos_airpods.png \
    icons/256x256/sfos_airpods.png

SAILFISHAPP_ICONS = 86x86 108x108 128x128 172x172 256x256
