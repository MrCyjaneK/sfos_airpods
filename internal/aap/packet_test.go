package aap

import (
	"encoding/hex"
	"testing"
)

func TestParseBatteryExample(t *testing.T) {
	raw := mustHex(t, "04000400040003020164020104016301010801110201")
	var st State
	st.Parse(raw)
	if st.Right == nil || st.Right.Level != 100 || st.Right.Status != "discharging" {
		t.Fatalf("right = %+v", st.Right)
	}
	if st.Left == nil || st.Left.Level != 99 || st.Left.Status != "charging" {
		t.Fatalf("left = %+v", st.Left)
	}
	if st.Case == nil || st.Case.Level != 17 || st.Case.Status != "discharging" {
		t.Fatalf("case = %+v", st.Case)
	}
}

func TestParseNoiseAndIgnoreConversationalAwareness(t *testing.T) {
	var st State
	st.Parse(mustHex(t, "0400040009002801000000"))
	if st.Noise != nil {
		t.Fatalf("conversational awareness stored as noise %02x", *st.Noise)
	}
	st.Parse(mustHex(t, "0400040009000d03000000"))
	if st.Noise == nil || *st.Noise != ModeTransparency {
		t.Fatalf("noise = %v", st.Noise)
	}
	if ModeName(*st.Noise) != "transparency" {
		t.Fatalf("name = %s", ModeName(*st.Noise))
	}
}

func TestParseEars(t *testing.T) {
	var st State
	st.Parse(mustHex(t, "0400040006000001"))
	if st.EarLeft == nil || *st.EarLeft != "in" || st.EarRight == nil || *st.EarRight != "out" {
		t.Fatalf("ears = %v %v", st.EarLeft, st.EarRight)
	}
}

func TestNoisePacket(t *testing.T) {
	got := hex.EncodeToString(NoisePacket(ModeANC))
	if got != "0400040009000d02000000" {
		t.Fatalf("packet = %s", got)
	}
}

func TestParseMode(t *testing.T) {
	mode, err := ParseMode("adaptive")
	if err != nil || mode != ModeAdaptive {
		t.Fatalf("mode = %02x err = %v", mode, err)
	}
	if _, err := ParseMode("loud"); err == nil {
		t.Fatal("expected error")
	}
}

func TestShortPacketsDoNotPanic(t *testing.T) {
	var st State
	st.Parse(nil)
	st.Parse(mustHex(t, "0400040004"))
	st.Parse(mustHex(t, "04000400040003"))
	st.Parse(mustHex(t, "010004000100"))
	if st.Any() {
		t.Fatalf("short packets produced %+v", st)
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
