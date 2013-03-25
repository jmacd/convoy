package geo

import "fmt"
import "math"

// Represent points in 3 dimensions, scaled to slightly larger than
// the size of Earth
const (
	EarthRadius = 6371000  // Meters
	EarthScale  = 6500000  // Meters

	surfaceFrac = float64(EarthRadius) / float64(EarthScale)
)

// ScaledPt is a unit of EarthScale / 2^31, which has approximately
// 3mm resolution at the earth's surface.
type EarthLoc int32
type Coords []EarthLoc  // TODO(jmacd) How would performance be if
			// this was [3]EarthLoc

// Square of distance between two points; actual distance is a chord
// on great circle.
type compDistance uint64

var infiniteDistance = compDistance(math.MaxUint64)

func (p0 Coords) Equals(p1 Coords) bool {
	return p0[0] == p1[0] && p0[1] == p1[1] && p0[2] == p1[2]
}

func degreeToRad(deg float64) float64 {
	if (deg < -180.0 || deg >= 180.0) {
		panic(fmt.Sprintf("Degree out of range: %.12f", deg))
	}
	return deg * math.Pi / 180.0
}

// Converts Lat/Long in degrees to scaled 3-d earth points.
func LatLongDegreesToCoords(lat, long float64, c Coords) {
	latRad, longRad := degreeToRad(lat), degreeToRad(long)
	x1 := math.Cos(latRad) * math.Cos(longRad)
	y1 := math.Cos(latRad) * math.Sin(longRad)
	z1 := math.Sin(latRad)
	c[0] = EarthLoc(x1 * surfaceFrac * math.MaxInt32)
	c[1] = EarthLoc(y1 * surfaceFrac * math.MaxInt32)
	c[2] = EarthLoc(z1 * surfaceFrac * math.MaxInt32)
}

func square(x EarthLoc) compDistance {
	return compDistance(x) * compDistance(x)
}

func comparableDistance(p0, p1 Coords) compDistance {
	return square(p0[0] - p1[0]) +
		square(p0[1] - p1[1]) +
		square(p0[2] - p1[2])
}
