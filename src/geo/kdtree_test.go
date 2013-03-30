package geo

import "testing"
import "fmt"
import "math"
import "math/rand"
import "runtime"

type testVertices []Vertex

type testNode struct {
	coord       [3]EarthLoc
	left, right Vertex
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (tv testVertices) Count() int {
	return len(tv)
}

func (tv testVertices) Node(i int) Vertex {
	return tv[i]
}

func (tn *testNode) Point() Coords {
	return tn.coord[:]
}

func (tn *testNode) String() string {
	return fmt.Sprintf("(%v,%v,%v)",
		tn.coord[0], tn.coord[1], tn.coord[2])
}

func (n *testNode) Left(_ Graph) Vertex {
	return n.left
}

func (n *testNode) Right(_ Graph) Vertex {
	return n.right
}

func (n *testNode) SetLeft(_ Graph, l Vertex) {
	n.left = l
}

func (n *testNode) SetRight(_ Graph, r Vertex) {
	n.right = r
}

func checkSorted(nodeX, nodeY, nodeZ Vertices, t *testing.T) bool {
	x, y, z := EarthLoc(math.MinInt32),
		EarthLoc(math.MinInt32),
		EarthLoc(math.MinInt32)
	for i, _ := range nodeX {
		if x > nodeX[i].Point()[0] {
			return false
		}
		if y > nodeY[i].Point()[1] {
			return false
		}
		if z > nodeZ[i].Point()[2] {
			return false
		}
		x = nodeX[i].Point()[0]
		y = nodeY[i].Point()[1]
		z = nodeZ[i].Point()[2]
	}
	return true
}

func TestMergeSort(t *testing.T) {
	const N = conSizeLimit * 3
	nodeX := make(Vertices, N)
	nodeY := make(Vertices, N)
	nodeZ := make(Vertices, N)
	for i, _ := range nodeX {
		tn := &testNode{}
		tn.coord[0] = EarthLoc(rand.Int31())
		tn.coord[1] = EarthLoc(rand.Int31())
		tn.coord[2] = EarthLoc(rand.Int31())
		nodeX[i] = tn
		nodeY[i] = tn
		nodeZ[i] = tn
	}
	if checkSorted(nodeX, nodeY, nodeZ, t) {
		t.Errorf("Improbable sorted inputs!")
	}
	nodeX = concurrentSort(nodeX, sortByX{})
	nodeY = concurrentSort(nodeY, sortByY{})
	nodeZ = concurrentSort(nodeZ, sortByZ{})
	if !checkSorted(nodeX, nodeY, nodeZ, t) {
		t.Errorf("Non-sorted outputs!")
	}
}

func testCoords(x, y, z int32) Coords {
	return []EarthLoc{EarthLoc(x), EarthLoc(y), EarthLoc(z)}
}
func testPoint(x, y, z int32) Vertex {
	tn := &testNode{}
	copy(tn.coord[:], testCoords(x, y, z))
	return tn
}

func TestTree(t *testing.T) {
	g := testVertices{
		testPoint(2, 3, 5),
		testPoint(5, 4, 4),
		testPoint(4, 7, 6),
		testPoint(7, 2, 3),
		testPoint(8, 1, 2),
		testPoint(9, 6, 1),
	}
	tree := NewTree(g)
	tree.Build()
	for _, v := range g {
		f := tree.FindExact(v.Point())
		if v != f {
			t.Errorf("Found %v not %v", f, v)
		}
	}
	origin := testCoords(0, 0, 0)
	near := tree.FindNearest(origin)
	if !near.Point().Equals(testCoords(2, 3, 5)) {
		t.Errorf("Nearest point failed: %s", near)
	}
}
