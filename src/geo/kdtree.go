package geo

import "log"
import "sort"

import "common"

const (
	conSizeLimit = 10000
)

type Vertices []Vertex

// Tree is a K-D Tree with K=3
type Tree struct {
	graph Graph
	root  Vertex
}

type Graph interface {
	Count() int
	Node(i int) Vertex
}

type Vertex interface {
	Point() Coords
	Left(Graph) Vertex
	Right(Graph) Vertex
	SetLeft(Graph, Vertex)
	SetRight(Graph, Vertex)
	String() string
}

func NewTree(graph Graph) *Tree {
	return &Tree{graph, nil}
}

func (n Vertices) Len() int      { return len(n) }
func (n Vertices) Swap(i, j int) { n[i], n[j] = n[j], n[i] }

type sorter interface {
	Interface(n Vertices) sort.Interface
	Less(c0, c1 Coords) bool
	Value(c0 Coords) EarthLoc
	String() string
}

type sortByX struct{}
type sortByY struct{}
type sortByZ struct{}

var xyzSorters = []sorter{sortByX{}, sortByY{}, sortByZ{}}

type byX struct{ Vertices }
type byY struct{ Vertices }
type byZ struct{ Vertices }

func (bx byX) Less(i, j int) bool {
	return bx.Vertices[i].Point()[0] < bx.Vertices[j].Point()[0]
}
func (by byY) Less(i, j int) bool {
	return by.Vertices[i].Point()[1] < by.Vertices[j].Point()[1]
}
func (by byZ) Less(i, j int) bool {
	return by.Vertices[i].Point()[2] < by.Vertices[j].Point()[2]
}

func (sortByX) Interface(n Vertices) sort.Interface { return byX{n} }
func (sortByY) Interface(n Vertices) sort.Interface { return byY{n} }
func (sortByZ) Interface(n Vertices) sort.Interface { return byZ{n} }

func (sortByX) Less(c0, c1 Coords) bool {
	return c0[0] < c1[0]
}
func (sortByY) Less(c0, c1 Coords) bool {
	return c0[1] < c1[1]
}
func (sortByZ) Less(c0, c1 Coords) bool {
	return c0[2] < c1[2]
}

func (sortByX) Value(c Coords) EarthLoc { return c[0] }
func (sortByY) Value(c Coords) EarthLoc { return c[1] }
func (sortByZ) Value(c Coords) EarthLoc { return c[2] }

func (sortByX) String() string { return "X" }
func (sortByY) String() string { return "Y" }
func (sortByZ) String() string { return "Z" }

func mergeSort(output, input Vertices,
	s sorter, con *common.Concurrentizer) {
	if len(input) < conSizeLimit {
		copy(output, input)
		sort.Sort(s.Interface(output))
		return
	}
	m := (len(input)-1)/2 + 1
	o0, o1 := output[0:m], output[m:]
	i0, i1 := input[0:m], input[m:]
	con.Do(len(o0), func(ccon *common.Concurrentizer) {
		mergeSort(o0, i0, s, ccon)
	})
	con.Do(len(o1), func(ccon *common.Concurrentizer) {
		mergeSort(o1, i1, s, ccon)
	})
	con.Wait()
	copy(input, output)
	merge(output, i0, i1, s)
}

func merge(out, in0, in1 Vertices, s sorter) {
	i, j, k := 0, 0, 0
	for ; i < len(in0) && j < len(in1); k++ {
		if s.Less(in0[i].Point(), in1[j].Point()) {
			out[k] = in0[i]
			i++
		} else {
			out[k] = in1[j]
			j++
		}
	}
	if i < len(in0) {
		copy(out[k:], in0[i:])
	} else if j < len(in1) {
		copy(out[k:], in1[j:])
	}
}

func concurrentSort(input Vertices, s sorter) Vertices {
	con := common.NewConcurrentizer(conSizeLimit)
	output := make(Vertices, len(input))
	con.Do(len(output), func(ccon *common.Concurrentizer) {
		mergeSort(output, input, s, ccon)
	}).Wait()
	return output
}

func (t *Tree) Build() {
	count := t.graph.Count()
	xdim := make(Vertices, count)
	ydim := make(Vertices, count)
	zdim := make(Vertices, count)
	for i := 0; i < count; i++ {
		v := t.graph.Node(i)
		xdim[i] = v
		ydim[i] = v
		zdim[i] = v
	}
	xdim = concurrentSort(xdim, sortByX{})
	ydim = concurrentSort(ydim, sortByY{})
	zdim = concurrentSort(zdim, sortByZ{})
	log.Println("Node-sorting finished")
	common.PrintMem()
	tmp := make(Vertices, count)
	con := common.NewConcurrentizer(conSizeLimit)
	con.Do(count, func(ccon *common.Concurrentizer) {
		t.root = t.buildTree(xdim, ydim, zdim, tmp,
			sortByX{}, sortByY{}, sortByZ{}, ccon)
	}).Wait()
}

func (t *Tree) FindExact(point Coords) Vertex {
	v := t.root
	for d := 0; ; d++ {
		s := xyzSorters[d%3]
		if point.Equals(v.Point()) {
			return v
		}
		if s.Less(point, v.Point()) {
			v = v.Left(t.graph)
		} else {
			v = v.Right(t.graph)
		}
	}
	return nil
}

// findMedian ensures that the midpoint is a true split, i.e., it is
// a lower bound of some point in this dimension.
func findMedian(dim Vertices, s sorter) int {
	mid := (len(dim)+1)/2 - 1
	for mid > 0 {
		if !s.Less(dim[mid-1].Point(), dim[mid].Point()) {
			mid--
		} else {
			break
		}
	}
	return mid
}

// Note: avoid passing dimN and sortN as slices because that
// causes them to escape and allocate a lot of memory.
func (t *Tree) buildTree(dim0, dim1, dim2, dimt Vertices,
	sort0, sort1, sort2 sorter,
	con *common.Concurrentizer) Vertex {
	if len(dim0) == 0 {
		return nil
	}
	// Choose the median point in dim0.
	mid := findMedian(dim0, sort0)
	split := dim0[mid]

	nextLeft := []Vertices{nil, nil, dim0[0:mid]}
	nextRight := []Vertices{nil, nil, dim0[mid+1:]}
	dims := []Vertices{dim0, dim1, dim2}

	tmpLeft := dimt[0:mid]
	tmpRight := dimt[mid+1:]

	for s := 1; s < len(dims); s++ {
		beingSplit := dims[s]

		// Split dims[s] into two halves in this plane
		for i, l, r := 0, 0, 0; i < len(dim0); i++ {
			p := beingSplit[i]
			if p == split {
				continue
			}
			if sort0.Less(p.Point(), split.Point()) {
				tmpLeft[l] = p
				l++
			} else {
				tmpRight[r] = p
				r++
			}
		}
		nextLeft[s-1] = tmpLeft
		nextRight[s-1] = tmpRight
		tmpLeft = beingSplit[0:mid]
		tmpRight = beingSplit[mid+1:]
	}

	con.Do(len(tmpLeft), func(ccon *common.Concurrentizer) {
		split.SetLeft(t.graph,
			t.buildTree(nextLeft[0], nextLeft[1], nextLeft[2],
				tmpLeft, sort1, sort2, sort0, ccon))
	})
	con.Do(len(tmpRight), func(ccon *common.Concurrentizer) {
		split.SetRight(t.graph,
			t.buildTree(nextRight[0], nextRight[1], nextRight[2],
				tmpRight, sort1, sort2, sort0, ccon))
	})
	con.Wait()
	return split
}

func (t *Tree) FindNearest(point Coords) Vertex {
	node, _ := t.findNearestPoint(point, t.root,
		sortByX{}, sortByY{}, sortByZ{}, 0)
	return node
}

func (t *Tree) findNearestPoint(point Coords, node Vertex,
	sort0, sort1, sort2 sorter, depth int) (Vertex, compDistance) {
	lc := node.Left(t.graph)
	rc := node.Right(t.graph)
	np := node.Point()
	pd := comparableDistance(point, np)
	if lc == nil && rc == nil {
		return node, pd
	}
	lessThan := sort0.Less(point, np)
	var closer, farther Vertex
	if lessThan {
		closer, farther = lc, rc
	} else {
		closer, farther = rc, lc
	}
	var nearest Vertex
	bisectDistance, distance := infiniteDistance, infiniteDistance
	if closer != nil {
		nearest, distance = t.findNearestPoint(
			point, closer, sort1, sort2, sort0, depth+1)
		bisectDistance =
			squareEarthLoc(sort0.Value(nearest.Point()) -
				sort0.Value(point))
	}
	// TODO(jmacd) Feels like this should be > half of the time
	if farther != nil && distance >= bisectDistance {
		fnear, fdist := t.findNearestPoint(
			point, farther, sort1, sort2, sort0, depth+1)
		if fdist < distance {
			nearest, distance = fnear, fdist
		}
	}
	if pd < distance {
		nearest, distance = node, pd
	}
	return nearest, distance
}
