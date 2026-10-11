// Package aap speaks the small slice of Apple's AirPods control protocol
// needed to read battery, noise mode, and ear state, and to set noise mode.
package aap

import (
	"encoding/binary"
	"fmt"
)

const PSM = 0x1001

// Documented component identifiers inside a battery notification.
const (
	batteryRight byte = 0x02
	batteryLeft  byte = 0x04
	batteryCase  byte = 0x08
)

// Documented noise-mode identifiers.
const (
	ModeOff          byte = 0x01
	ModeANC          byte = 0x02
	ModeTransparency byte = 0x03
	ModeAdaptive     byte = 0x04
)

const (
	earIn   byte = 0x00
	earOut  byte = 0x01
	earCase byte = 0x02
)

var (
	handshakePacket = []byte{
		0x00, 0x00, 0x04, 0x00, 0x01, 0x00, 0x02, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	featuresPacket = []byte{
		0x04, 0x00, 0x04, 0x00, 0x4d, 0x00, 0xff, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	notificationsPacket = []byte{
		0x04, 0x00, 0x04, 0x00, 0x0f, 0x00, 0xff, 0xff, 0xff, 0xff,
	}
)

// Battery is one AirPods component reported by a battery notification.
type Battery struct {
	Level  int
	Status string
}

// State is the control data gathered from one session.
type State struct {
	Left, Right, Case *Battery
	Noise             *byte
	EarLeft, EarRight *string
}

func (s State) Any() bool {
	return s.Left != nil || s.Right != nil || s.Case != nil || s.Noise != nil || s.EarLeft != nil || s.EarRight != nil
}

func (s State) complete() bool {
	return s.Noise != nil && s.EarLeft != nil && s.EarRight != nil && (s.Left != nil || s.Right != nil)
}

// ParseMode accepts the CLI noise-mode names.
func ParseMode(name string) (byte, error) {
	switch name {
	case "off":
		return ModeOff, nil
	case "anc":
		return ModeANC, nil
	case "transparency":
		return ModeTransparency, nil
	case "adaptive":
		return ModeAdaptive, nil
	default:
		return 0, fmt.Errorf("unknown noise mode %q (want off, anc, transparency, or adaptive)", name)
	}
}

// ModeName is the CLI name for a noise-mode byte.
func ModeName(mode byte) string {
	switch mode {
	case ModeOff:
		return "off"
	case ModeANC:
		return "anc"
	case ModeTransparency:
		return "transparency"
	case ModeAdaptive:
		return "adaptive"
	default:
		return fmt.Sprintf("unknown(%02x)", mode)
	}
}

// NoisePacket is the command that sets the listening mode.
func NoisePacket(mode byte) []byte {
	return []byte{0x04, 0x00, 0x04, 0x00, 0x09, 0x00, 0x0d, mode, 0x00, 0x00, 0x00}
}

// Parse folds one L2CAP payload into state. Packets for other features are ignored.
func (s *State) Parse(p []byte) {
	if len(p) < 6 || p[0] != 0x04 || p[1] != 0x00 || p[2] != 0x04 || p[3] != 0x00 {
		return
	}
	opcode := binary.LittleEndian.Uint16(p[4:6])
	body := p[6:]
	switch opcode {
	case 0x0004:
		s.parseBattery(body)
	case 0x0006:
		s.parseEars(body)
	case 0x0009:
		s.parseControl(body)
	}
}

func (s *State) parseBattery(body []byte) {
	if len(body) < 1 {
		return
	}
	count := int(body[0])
	off := 1
	for i := 0; i < count; i++ {
		if off+5 > len(body) {
			return
		}
		component := body[off]
		level := int(body[off+2])
		status := batteryStatus(body[off+3])
		bat := &Battery{Level: level, Status: status}
		switch component {
		case batteryLeft:
			s.Left = bat
		case batteryRight:
			s.Right = bat
		case batteryCase:
			s.Case = bat
		}
		off += 5
	}
}

func batteryStatus(b byte) string {
	switch b {
	case 0x00:
		return "unknown"
	case 0x01:
		return "charging"
	case 0x02:
		return "discharging"
	case 0x04:
		return "disconnected"
	default:
		return fmt.Sprintf("unknown(%02x)", b)
	}
}

func (s *State) parseEars(body []byte) {
	if len(body) < 2 {
		return
	}
	// First byte is the left bud, second is the right.
	left := earName(body[0])
	right := earName(body[1])
	s.EarLeft = &left
	s.EarRight = &right
}

func earName(b byte) string {
	switch b {
	case earIn:
		return "in"
	case earOut:
		return "out"
	case earCase:
		return "case"
	default:
		return fmt.Sprintf("unknown(%02x)", b)
	}
}

func (s *State) parseControl(body []byte) {
	// 0x0d is noise control. 0x28 is conversational awareness and must not
	// be reported as a listening mode.
	if len(body) < 2 || body[0] != 0x0d {
		return
	}
	mode := body[1]
	s.Noise = &mode
}

// FormatStatus is the CLI's stable text output.
func FormatStatus(name, address string, st State) string {
	return fmt.Sprintf(
		"name: %s\naddress: %s\nbattery-left: %s\nbattery-right: %s\nbattery-case: %s\nnoise: %s\near-left: %s\near-right: %s\n",
		name,
		address,
		formatBattery(st.Left),
		formatBattery(st.Right),
		formatBattery(st.Case),
		formatNoise(st.Noise),
		formatEar(st.EarLeft),
		formatEar(st.EarRight),
	)
}

func formatBattery(b *Battery) string {
	if b == nil {
		return "unknown"
	}
	return fmt.Sprintf("%d%% %s", b.Level, b.Status)
}

func formatNoise(mode *byte) string {
	if mode == nil {
		return "unknown"
	}
	return ModeName(*mode)
}

func formatEar(ear *string) string {
	if ear == nil {
		return "unknown"
	}
	return *ear
}
