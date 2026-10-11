package bluez

import "testing"

func TestIsAirPods(t *testing.T) {
	if !isAirPods("AirPods Pro", "", "") {
		t.Fatal("name")
	}
	if !isAirPods("Headphones", "bluetooth:v004Cp2024d215C", "audio-headphones") {
		t.Fatal("modalias")
	}
	if isAirPods("Mouse", "bluetooth:v004Cp2024d215C", "input-mouse") {
		t.Fatal("non-audio apple device")
	}
	if isAirPods("Other Buds", "bluetooth:v000Ap0001d0001", "audio-headphones") {
		t.Fatal("other vendor")
	}
}
