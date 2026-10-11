// Package bluez finds a connected AirPods device on the system BlueZ bus.
package bluez

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

// Device is an AirPods unit and the adapter it is using.
type Device struct {
	Path           dbus.ObjectPath
	Name           string
	Address        string
	AdapterAddress string
}

// FindConnected returns the one connected AirPods device.
func FindConnected() (Device, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return Device{}, fmt.Errorf("system bus: %w", err)
	}
	defer conn.Close()

	found, err := connected(conn)
	if err != nil {
		return Device{}, err
	}
	switch len(found) {
	case 0:
		return Device{}, fmt.Errorf("no connected AirPods")
	case 1:
		if found[0].Address == "" {
			return Device{}, fmt.Errorf("connected AirPods have no address")
		}
		return found[0], nil
	default:
		var list []string
		for _, dev := range found {
			list = append(list, dev.Name+" "+dev.Address)
		}
		return Device{}, fmt.Errorf("more than one connected AirPods: %s", strings.Join(list, ", "))
	}
}

func connected(conn *dbus.Conn) ([]Device, error) {
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err := conn.Object("org.bluez", "/").Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&objects)
	if err != nil {
		return nil, fmt.Errorf("list bluetooth devices: %w", err)
	}
	var found []Device
	for path, ifaces := range objects {
		dev, ok := ifaces["org.bluez.Device1"]
		if !ok || !variantBool(dev["Connected"]) || !isAirPods(variantString(dev["Name"]), variantString(dev["Modalias"]), variantString(dev["Icon"])) {
			continue
		}
		adapter := variantPath(dev["Adapter"])
		found = append(found, Device{
			Path:           path,
			Name:           variantString(dev["Name"]),
			Address:        variantString(dev["Address"]),
			AdapterAddress: variantString(objects[adapter]["org.bluez.Adapter1"]["Address"]),
		})
	}
	return found, nil
}

func isAirPods(name, modalias, icon string) bool {
	if strings.Contains(strings.ToLower(name), "airpods") {
		return true
	}
	return strings.Contains(strings.ToLower(modalias), "v004c") && icon == "audio-headphones"
}

func variantString(v dbus.Variant) string {
	s, _ := v.Value().(string)
	return s
}

func variantBool(v dbus.Variant) bool {
	b, _ := v.Value().(bool)
	return b
}

func variantPath(v dbus.Variant) dbus.ObjectPath {
	switch p := v.Value().(type) {
	case dbus.ObjectPath:
		return p
	case string:
		return dbus.ObjectPath(p)
	default:
		return ""
	}
}
