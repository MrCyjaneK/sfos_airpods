package aap

// CaseView is the case battery published to clients.
// Level -1 means no live reading has been seen yet.
type CaseView struct {
	Level  int
	Status string
	Live   bool
}

// CaseMemory keeps the last charging or discharging case sample.
// A disconnected sample does not replace it.
type CaseMemory struct {
	level int
	ok    bool
}

// Observe applies one case sample. A nil sample publishes the saved level.
func (m *CaseMemory) Observe(b *Battery) CaseView {
	if b != nil && (b.Status == "charging" || b.Status == "discharging") {
		m.level = b.Level
		m.ok = true
		return CaseView{Level: b.Level, Status: b.Status, Live: true}
	}
	status := "disconnected"
	if b != nil && b.Status != "" {
		status = b.Status
	}
	if m.ok {
		return CaseView{Level: m.level, Status: status, Live: false}
	}
	if b == nil {
		return CaseView{Level: -1, Status: "", Live: false}
	}
	return CaseView{Level: -1, Status: status, Live: false}
}
