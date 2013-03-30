package geo

import "testing"

type coordCheck struct {
	in     string
	degree float64
}

var coordExpect = []coordCheck{
	{"10°N", 10.0},
	{"10.5°N", 10.5},
	{"89.999°S", -89.999},
	{"10.123°E", 10.123},
	{"0.3°W", -0.3},
	{"30°30′W", -30.5},
	{"30°30.5′W", -30.5 - (.5 / 60.0)},
	{"30°30′30″E", 30.5 + (30 / 3600.0)},
}

func TestToDegrees(t *testing.T) {
	for _, ce := range coordExpect {
		d := StringToDegrees(ce.in)
		if d != ce.degree {
			t.Errorf("Expected %s -> %.10f got %.10f", ce.in, ce.degree, d)
		}
	}
}
