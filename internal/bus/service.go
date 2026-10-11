// Package bus exports the AirPods control object on the system bus.
package bus

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"

	"github.com/mrcyjanek/sfos_airpods/internal/aap"
)

const (
	// Name is the well-known system bus name.
	Name = "net.mrcyjanek.sfos_airpods"
	// Path is the object path.
	Path = "/net/mrcyjanek/sfos_airpods"
	// Iface is the application interface.
	Iface = "net.mrcyjanek.sfos_airpods"
)

// Link can send a noise-mode command on the open control channel.
type Link interface {
	SetNoise(mode byte) error
}

// Service is the state published to the app and the CLI.
type Service struct {
	props   *prop.Properties
	mu      sync.Mutex
	cases   aap.CaseMemory
	state   aap.State
	link    Link
	name    string
	address string
	up      bool
}

type methods struct {
	svc *Service
}

// Export publishes the object and takes the well-known bus name.
func Export(conn *dbus.Conn) (*Service, error) {
	svc := &Service{}
	props, err := prop.Export(conn, Path, prop.Map{
		Iface: {
			"Connected":          {Value: false, Emit: prop.EmitTrue},
			"Name":               {Value: "", Emit: prop.EmitTrue},
			"Address":            {Value: "", Emit: prop.EmitTrue},
			"BatteryLeft":        {Value: int32(-1), Emit: prop.EmitTrue},
			"BatteryLeftStatus":  {Value: "", Emit: prop.EmitTrue},
			"BatteryRight":       {Value: int32(-1), Emit: prop.EmitTrue},
			"BatteryRightStatus": {Value: "", Emit: prop.EmitTrue},
			"BatteryCase":        {Value: int32(-1), Emit: prop.EmitTrue},
			"BatteryCaseLive":    {Value: false, Emit: prop.EmitTrue},
			"BatteryCaseStatus":  {Value: "", Emit: prop.EmitTrue},
			"Noise":              {Value: "", Emit: prop.EmitTrue},
			"EarLeft":            {Value: "", Emit: prop.EmitTrue},
			"EarRight":           {Value: "", Emit: prop.EmitTrue},
		},
	})
	if err != nil {
		return nil, err
	}
	svc.props = props
	if err := conn.Export(&methods{svc: svc}, Path, Iface); err != nil {
		return nil, err
	}
	reply, err := conn.RequestName(Name, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf("bus name %s is already taken", Name)
	}
	return svc, nil
}

func (m *methods) SetNoiseMode(mode string) *dbus.Error {
	return m.svc.setNoise(mode)
}

// Attach makes link the target of SetNoiseMode until Detach.
func (s *Service) Attach(link Link) {
	s.mu.Lock()
	s.link = link
	s.mu.Unlock()
}

// Detach drops link if it is still the active one.
func (s *Service) Detach(link Link) {
	s.mu.Lock()
	if s.link == link {
		s.link = nil
	}
	s.mu.Unlock()
}

// MarkConnected publishes the device as connected.
func (s *Service) MarkConnected(name, address string) {
	s.mu.Lock()
	s.up = true
	s.name = name
	s.address = address
	s.publish()
	s.mu.Unlock()
}

// MarkDisconnected keeps the last battery readings and case fallback.
func (s *Service) MarkDisconnected() {
	s.mu.Lock()
	s.up = false
	s.link = nil
	s.state.Noise = nil
	s.state.EarLeft = nil
	s.state.EarRight = nil
	s.state.Case = nil
	s.publish()
	s.mu.Unlock()
}

// OnPacket folds one control packet into the published properties.
func (s *Service) OnPacket(p []byte) {
	s.mu.Lock()
	s.state.Parse(p)
	s.publish()
	s.mu.Unlock()
}

func (s *Service) setNoise(mode string) *dbus.Error {
	want, err := aap.ParseMode(mode)
	if err != nil {
		return dbus.MakeFailedError(err)
	}
	s.mu.Lock()
	link := s.link
	s.mu.Unlock()
	if link == nil {
		return dbus.MakeFailedError(errors.New("not connected"))
	}
	if err := link.SetNoise(want); err != nil {
		return dbus.MakeFailedError(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		s.mu.Lock()
		match := s.state.Noise != nil && *s.state.Noise == want
		if s.state.Noise != nil {
			got = aap.ModeName(*s.state.Noise)
		} else {
			got = "unknown"
		}
		s.mu.Unlock()
		if match {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return dbus.MakeFailedError(fmt.Errorf("noise mode is %s, wanted %s", got, mode))
}

func (s *Service) publish() {
	s.setBool("Connected", s.up)
	s.setString("Name", s.name)
	s.setString("Address", s.address)
	s.setLevel("BatteryLeft", s.state.Left)
	s.setString("BatteryLeftStatus", statusName(s.state.Left))
	s.setLevel("BatteryRight", s.state.Right)
	s.setString("BatteryRightStatus", statusName(s.state.Right))
	view := s.cases.Observe(s.state.Case)
	s.setInt("BatteryCase", int32(view.Level))
	s.setBool("BatteryCaseLive", view.Live)
	s.setString("BatteryCaseStatus", view.Status)
	s.setString("Noise", noiseName(s.state.Noise))
	s.setString("EarLeft", earName(s.state.EarLeft))
	s.setString("EarRight", earName(s.state.EarRight))
}

func statusName(b *aap.Battery) string {
	if b == nil {
		return ""
	}
	return b.Status
}

func (s *Service) setLevel(name string, b *aap.Battery) {
	level := int32(-1)
	if b != nil {
		level = int32(b.Level)
	}
	s.setInt(name, level)
}

func noiseName(mode *byte) string {
	if mode == nil {
		return ""
	}
	return aap.ModeName(*mode)
}

func earName(ear *string) string {
	if ear == nil {
		return ""
	}
	return *ear
}

func (s *Service) setBool(name string, v bool) {
	if s.props.GetMust(Iface, name).(bool) == v {
		return
	}
	s.props.SetMust(Iface, name, v)
}

func (s *Service) setString(name, v string) {
	if s.props.GetMust(Iface, name).(string) == v {
		return
	}
	s.props.SetMust(Iface, name, v)
}

func (s *Service) setInt(name string, v int32) {
	if s.props.GetMust(Iface, name).(int32) == v {
		return
	}
	s.props.SetMust(Iface, name, v)
}
