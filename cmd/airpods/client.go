package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"

	"github.com/mrcyjanek/sfos_airpods/internal/aap"
	"github.com/mrcyjanek/sfos_airpods/internal/bus"
)

var errUsage = errors.New("usage")

func run(args []string) error {
	if len(args) == 0 {
		return errUsage
	}
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("system bus: %w", err)
	}
	defer conn.Close()
	obj := conn.Object(bus.Name, dbus.ObjectPath(bus.Path))

	switch args[0] {
	case "status":
		if len(args) != 1 {
			return errUsage
		}
	case "noise":
		if len(args) != 2 {
			return errUsage
		}
		if _, err := aap.ParseMode(args[1]); err != nil {
			return err
		}
		if err := callError(obj.Call(bus.Iface+".SetNoiseMode", 0, args[1]).Err); err != nil {
			return err
		}
	default:
		return errUsage
	}
	props, err := getAll(obj)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, format(props))
	return nil
}

func getAll(obj dbus.BusObject) (map[string]dbus.Variant, error) {
	var props map[string]dbus.Variant
	err := obj.Call("org.freedesktop.DBus.Properties.GetAll", 0, bus.Iface).Store(&props)
	if err != nil {
		return nil, callError(err)
	}
	return props, nil
}

func callError(err error) error {
	if err == nil {
		return nil
	}
	var derr dbus.Error
	if errors.As(err, &derr) {
		switch derr.Name {
		case "org.freedesktop.DBus.Error.ServiceUnknown", "org.freedesktop.DBus.Error.NameHasNoOwner":
			return errors.New("daemon is not running")
		}
	}
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 {
		msg = msg[i+2:]
	}
	return errors.New(msg)
}

func format(props map[string]dbus.Variant) string {
	connected := "no"
	if propBool(props["Connected"]) {
		connected = "yes"
	}
	left, leftOK := propInt(props["BatteryLeft"])
	right, rightOK := propInt(props["BatteryRight"])
	caseLevel, caseOK := propInt(props["BatteryCase"])
	return fmt.Sprintf(
		"connected: %s\nname: %s\naddress: %s\nbattery-left: %s\nbattery-right: %s\nbattery-case: %s\nnoise: %s\near-left: %s\near-right: %s\n",
		connected,
		propText(props["Name"]),
		propText(props["Address"]),
		formatLevel(left, leftOK),
		formatLevel(right, rightOK),
		formatCase(caseLevel, caseOK, propString(props["BatteryCaseStatus"])),
		propText(props["Noise"]),
		propText(props["EarLeft"]),
		propText(props["EarRight"]),
	)
}

func formatLevel(n int, ok bool) string {
	if !ok || n < 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d%%", n)
}

func formatCase(n int, ok bool, status string) string {
	if !ok || n < 0 {
		return "unknown"
	}
	if status == "" {
		return fmt.Sprintf("%d%%", n)
	}
	return fmt.Sprintf("%d%% %s", n, status)
}

func propText(v dbus.Variant) string {
	s := propString(v)
	if s == "" {
		return "unknown"
	}
	return s
}

func propString(v dbus.Variant) string {
	s, _ := v.Value().(string)
	return s
}

func propBool(v dbus.Variant) bool {
	b, _ := v.Value().(bool)
	return b
}

func propInt(v dbus.Variant) (int, bool) {
	switch n := v.Value().(type) {
	case int32:
		return int(n), true
	case int16:
		return int(n), true
	case int64:
		return int(n), true
	case uint32:
		return int(n), true
	default:
		return 0, false
	}
}
