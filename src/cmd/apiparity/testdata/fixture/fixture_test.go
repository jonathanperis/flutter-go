package fixture

import "testing"

// The apiparity tests check which fixture declarations this file refers to.
func TestUses(t *testing.T) {
	p := NewPoint(1, 2)
	_ = p.Add(p)
	_ = RegistryCount
	_ = Widget{Key: "k"}
	var h Hue = HueRed
	_ = h.IsDark()
}
