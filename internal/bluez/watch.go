package bluez

import (
	"context"
	"log"

	"github.com/godbus/dbus/v5"
)

// Event is a change in whether an AirPods device is connected.
type Event struct {
	Connected bool
	Device    Device
}

// Watch blocks on BlueZ signals and reports AirPods connect and disconnect.
// The connection must stay open for as long as the caller reads the channel.
func Watch(ctx context.Context, conn *dbus.Conn) (<-chan Event, error) {
	if err := conn.AddMatchSignal(
		dbus.WithMatchSender("org.bluez"),
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
	); err != nil {
		return nil, err
	}
	if err := conn.AddMatchSignal(
		dbus.WithMatchSender("org.bluez"),
		dbus.WithMatchInterface("org.freedesktop.DBus.ObjectManager"),
		dbus.WithMatchMember("InterfacesAdded"),
	); err != nil {
		return nil, err
	}
	signals := make(chan *dbus.Signal, 32)
	conn.Signal(signals)
	out := make(chan Event, 4)
	go func() {
		defer close(out)
		var current dbus.ObjectPath
		devs, err := connected(conn)
		if err != nil {
			log.Printf("airpodsd: %v", err)
		} else if len(devs) == 1 {
			current = devs[0].Path
			send(ctx, out, Event{Connected: true, Device: devs[0]})
		} else if len(devs) > 1 {
			log.Printf("airpodsd: more than one connected AirPods")
		}
		for {
			select {
			case <-ctx.Done():
				return
			case sig, ok := <-signals:
				if !ok || sig == nil {
					return
				}
				ev, path, ok := interpret(conn, sig)
				if !ok {
					continue
				}
				if ev.Connected {
					current = path
				} else if path == current {
					current = ""
				} else {
					continue
				}
				send(ctx, out, ev)
			}
		}
	}()
	return out, nil
}

func send(ctx context.Context, out chan<- Event, ev Event) {
	select {
	case <-ctx.Done():
	case out <- ev:
	}
}

func interpret(conn *dbus.Conn, sig *dbus.Signal) (Event, dbus.ObjectPath, bool) {
	switch sig.Name {
	case "org.freedesktop.DBus.Properties.PropertiesChanged":
		return interpretProperties(conn, sig)
	case "org.freedesktop.DBus.ObjectManager.InterfacesAdded":
		return interpretAdded(conn, sig)
	default:
		return Event{}, "", false
	}
}

func interpretProperties(conn *dbus.Conn, sig *dbus.Signal) (Event, dbus.ObjectPath, bool) {
	if len(sig.Body) < 2 {
		return Event{}, "", false
	}
	iface, _ := sig.Body[0].(string)
	if iface != "org.bluez.Device1" {
		return Event{}, "", false
	}
	changed, _ := sig.Body[1].(map[string]dbus.Variant)
	connProp, ok := changed["Connected"]
	if !ok {
		return Event{}, "", false
	}
	up, _ := connProp.Value().(bool)
	if !up {
		return Event{Connected: false}, sig.Path, true
	}
	dev, ok := loadDevice(conn, sig.Path)
	if !ok {
		return Event{}, "", false
	}
	return Event{Connected: true, Device: dev}, sig.Path, true
}

func interpretAdded(conn *dbus.Conn, sig *dbus.Signal) (Event, dbus.ObjectPath, bool) {
	if len(sig.Body) < 1 {
		return Event{}, "", false
	}
	path, ok := sig.Body[0].(dbus.ObjectPath)
	if !ok {
		return Event{}, "", false
	}
	dev, ok := loadDevice(conn, path)
	if !ok {
		return Event{}, "", false
	}
	return Event{Connected: true, Device: dev}, path, true
}

func loadDevice(conn *dbus.Conn, path dbus.ObjectPath) (Device, bool) {
	var props map[string]dbus.Variant
	err := conn.Object("org.bluez", path).Call("org.freedesktop.DBus.Properties.GetAll", 0, "org.bluez.Device1").Store(&props)
	if err != nil {
		return Device{}, false
	}
	if !variantBool(props["Connected"]) || !isAirPods(variantString(props["Name"]), variantString(props["Modalias"]), variantString(props["Icon"])) {
		return Device{}, false
	}
	adapter := variantPath(props["Adapter"])
	var address dbus.Variant
	if adapter != "" {
		_ = conn.Object("org.bluez", adapter).Call("org.freedesktop.DBus.Properties.Get", 0, "org.bluez.Adapter1", "Address").Store(&address)
	}
	dev := Device{
		Path:           path,
		Name:           variantString(props["Name"]),
		Address:        variantString(props["Address"]),
		AdapterAddress: variantString(address),
	}
	if dev.Address == "" {
		return Device{}, false
	}
	return dev, true
}
