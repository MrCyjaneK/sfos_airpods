package aap

import "testing"

func TestCaseMemoryKeepsLastLiveSample(t *testing.T) {
	var m CaseMemory
	got := m.Observe(&Battery{Level: 0, Status: "disconnected"})
	if got.Level != -1 || got.Live || got.Status != "disconnected" {
		t.Fatalf("first disconnected = %+v", got)
	}
	got = m.Observe(&Battery{Level: 40, Status: "charging"})
	if got.Level != 40 || !got.Live || got.Status != "charging" {
		t.Fatalf("charging = %+v", got)
	}
	got = m.Observe(&Battery{Level: 0, Status: "disconnected"})
	if got.Level != 40 || got.Live || got.Status != "disconnected" {
		t.Fatalf("fallback = %+v", got)
	}
	got = m.Observe(nil)
	if got.Level != 40 || got.Live || got.Status != "disconnected" {
		t.Fatalf("nil = %+v", got)
	}
	got = m.Observe(&Battery{Level: 30, Status: "discharging"})
	if got.Level != 30 || !got.Live || got.Status != "discharging" {
		t.Fatalf("discharging = %+v", got)
	}
}

func TestCaseMemoryIgnoresUnknownStatus(t *testing.T) {
	var m CaseMemory
	m.Observe(&Battery{Level: 12, Status: "discharging"})
	got := m.Observe(&Battery{Level: 99, Status: "unknown"})
	if got.Level != 12 || got.Live || got.Status != "unknown" {
		t.Fatalf("unknown = %+v", got)
	}
}
