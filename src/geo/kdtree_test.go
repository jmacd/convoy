package geo

import "testing"
import "math"
import "math/rand"
import "runtime"

type testNode struct {
	coord [2]ScaledRad
}

func (tn *testNode) Coord() []ScaledRad {
	return tn.coord[:]
}

func checkSorted(nodeX, nodeY []Node, t *testing.T) bool {
	x, y := ScaledRad(math.MinInt32), ScaledRad(math.MinInt32)
	for i, _ := range nodeX {
		if x > nodeX[i].Coord()[0] {
			return false
		}
		if y > nodeY[i].Coord()[1] {
			return false
		}
		x = nodeX[i].Coord()[0]
		y = nodeY[i].Coord()[1]
	}
	return true
}

func TestMergeSort(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	const N = seqSortLimit * 3
	nodeX := make([]Node, N)
	nodeY := make([]Node, N)
	for i, _ := range nodeX {
		tn := &testNode{}
		tn.coord[0] = ScaledRad(rand.Int31())
		tn.coord[1] = ScaledRad(rand.Int31())
		nodeX[i] = tn
		nodeY[i] = tn
	}
	if checkSorted(nodeX, nodeY, t) {
		t.Errorf("Improbable sorted inputs!")
	}
	nodeX = concurrentSort(nodeX, sortByX{})
	nodeY = concurrentSort(nodeY, sortByY{})
	if !checkSorted(nodeX, nodeY, t) {
		t.Errorf("Non-sorted outputs!")
	}
}
