package geo

import "fmt"

// ScaledRad is a unit of Pi / 2^31 radians, which has approximately 1
// meter resolution at the earth's equator.
type ScaledRad int32

func ScaleDegrees(deg float64) ScaledRad {
	if (deg < -180.0 || deg >= 180.0) {
		panic(fmt.Sprintf("Degree out of range: %.12f", deg))
	}
	unit := (deg / 180.0)  // Range [-1,1)
	return ScaledRad(unit * float64(1 << 31))
}

