package geo

import "testing"
import "fmt"
import "math"
import "math/rand"
import "runtime"

type testNode struct {
	coord [2]ScaledRad
	left, right Vertex
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (tn *testNode) Point() Coords {
	return tn.coord[:]
}

func (tn *testNode) String() string {
	return fmt.Sprintf("(%v,%v)", tn.coord[0], tn.coord[1])
}

func (n *testNode) Left() Vertex {
	return n.left
}

func (n *testNode) Right() Vertex {
	return n.right
}

func (n *testNode) SetLeft(l Vertex) {
	n.left = l
}

func (n *testNode) SetRight(r Vertex) {
	n.right = r
}

func checkSorted(nodeX, nodeY Vertices, t *testing.T) bool {
	x, y := ScaledRad(math.MinInt32), ScaledRad(math.MinInt32)
	for i, _ := range nodeX {
		if x > nodeX[i].Point()[0] {
			return false
		}
		if y > nodeY[i].Point()[1] {
			return false
		}
		x = nodeX[i].Point()[0]
		y = nodeY[i].Point()[1]
	}
	return true
}

func TestMergeSort(t *testing.T) {
	const N = conSizeLimit * 3
	nodeX := make(Vertices, N)
	nodeY := make(Vertices, N)
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

func testPoint(x, y int) Vertex {
	return &testNode{[...]ScaledRad{ScaledRad(x),ScaledRad(y)}, nil, nil}
}

func TestTree(t *testing.T) {
	tree := NewTree()
	g := []Vertex{
		testPoint(2, 3),
		testPoint(5, 4),
		testPoint(4, 7),
		testPoint(7, 2),
		testPoint(8, 1),
		testPoint(9, 6),
	}
	tree.Build(g)
	for _, v := range g {
		f := tree.FindExact(v.Point())
		if v != f {
			t.Errorf("Found %v not %v", f, v)
		}
	}
}
