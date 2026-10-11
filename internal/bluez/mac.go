package bluez

import "fmt"

// ParseMAC reads a Bluetooth address in display order.
// An empty string returns the zero address.
func ParseMAC(s string) ([6]byte, error) {
	var addr [6]byte
	if s == "" {
		return addr, nil
	}
	n, err := fmt.Sscanf(s, "%02x:%02x:%02x:%02x:%02x:%02x", &addr[0], &addr[1], &addr[2], &addr[3], &addr[4], &addr[5])
	if err != nil || n != 6 {
		return addr, fmt.Errorf("parse %q: %w", s, err)
	}
	return addr, nil
}
