package geo

import "testing"

func TestGCD(t *testing.T) {
	var SFO, JFK [3]EarthLoc
	// Coordinates taken from Wikipedia
	SphereCoords{StringToDegrees("40°38′23″N"),
		StringToDegrees("73°46′44″W")}.ToCoords(JFK[:])

	SphereCoords{StringToDegrees("37°37′09″N"),
		StringToDegrees("122°22′31″W")}.ToCoords(SFO[:])

	dist := GreatCircleDistance(JFK[:], SFO[:])
	// SFO to JFK is 4161 according to gc.kls2.com, this says
	if dist < 4151000 || dist > 4152000 {
		t.Errorf("Wrong distance %.9f", dist)
	}
}
