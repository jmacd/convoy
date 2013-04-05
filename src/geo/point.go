package geo

import "fmt"
import "math"

import "common"

// Represent points in 3 dimensions, scaled to slightly larger than
// the size of Earth
const (
	earthRadius      = 6371000 // Meters
	earthDiameter    = float64(earthRadius * 2)
	earthPrecision   = float64(earthRadius) / float64(math.MaxInt32)
	invEarthDiameter = 1.0 / earthDiameter
)

// EarthLoc is a unit of earthPrecision, which has approximately 3mm
// resolution at the earth's surface.
type EarthLoc int32
type Coords []EarthLoc

type SphereCoords struct {
	Lat, Long float64 // In degrees
}

type CityStateLoc struct {
	common.CityState
	SphereCoords
}

// Square of distance between two points; actual distance is a chord
// on great circle.
type compDistance uint64

var infiniteDistance = compDistance(math.MaxUint64)

func (p0 Coords) Equals(p1 Coords) bool {
	return p0[0] == p1[0] && p0[1] == p1[1] && p0[2] == p1[2]
}

func (sc SphereCoords) Defined() bool {
	return sc.Lat != 0.0 && sc.Long != 0.0
}

func degreeToRad(deg float64) float64 {
	if deg < -180.0 || deg >= 180.0 {
		panic(fmt.Sprintf("Degree out of range: %.12f", deg))
	}
	return deg * math.Pi / 180.0
}

// Converts Lat/Long in degrees to scaled 3-d earth points.
func (sc SphereCoords) ToCoords(c Coords) {
	latRad, longRad := degreeToRad(sc.Lat), degreeToRad(sc.Long)
	x1 := math.Cos(latRad) * math.Cos(longRad)
	y1 := math.Cos(latRad) * math.Sin(longRad)
	z1 := math.Sin(latRad)
	c[0] = EarthLoc(x1 * math.MaxInt32)
	c[1] = EarthLoc(y1 * math.MaxInt32)
	c[2] = EarthLoc(z1 * math.MaxInt32)
}

func squareEarthLoc(x EarthLoc) compDistance {
	return compDistance(x) * compDistance(x)
}

func comparableDistance(p0, p1 Coords) compDistance {
	return squareEarthLoc(p0[0]-p1[0]) +
		squareEarthLoc(p0[1]-p1[1]) +
		squareEarthLoc(p0[2]-p1[2])
}

func squareRealLoc(x float64) float64 {
	return x * x
}

func chordLength(p0, p1 Coords) float64 {
	return math.Sqrt(
		squareRealLoc(float64(p0[0]-p1[0])*earthPrecision) +
			squareRealLoc(float64(p0[1]-p1[1])*earthPrecision) +
			squareRealLoc(float64(p0[2]-p1[2])*earthPrecision))
}

func GreatCircleDistance(p0, p1 Coords) float64 {
	a := chordLength(p0, p1) * invEarthDiameter
	if a > 1.0 {
		panic(fmt.Sprintln("Can't happen", a))
	}
	return earthDiameter * math.Asin(a)
}
