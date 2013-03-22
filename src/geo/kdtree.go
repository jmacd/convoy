package geo

import "sort"
import "log"

import "common"

const (
	conSizeLimit = 1000
)

type Vertices []Vertex

// Tree is a K-D Tree with K=2
type Tree struct {
	root Vertex
}

type Vertex interface {
	Point() Coords
	String() string
	Left() Vertex
	Right() Vertex
	SetLeft(Vertex)
	SetRight(Vertex)
}

func NewTree() *Tree {
	return &Tree{}
}

func (n Vertices) Len() int { return len(n) }
func (n Vertices) Swap(i, j int) { n[i], n[j] = n[j], n[i] }

type sorter interface {
	Interface(n Vertices) sort.Interface
	Less(c0, c1 Coords) bool
}

type sortByX struct {}
type sortByY struct {}

type byX struct { Vertices }
type byY struct { Vertices }

func (bx byX) Less(i, j int) bool { 
	return bx.Vertices[i].Point()[0] < bx.Vertices[j].Point()[0] 
}
func (by byY) Less(i, j int) bool { 
	return by.Vertices[i].Point()[1] < by.Vertices[j].Point()[1] 
}

func (sortByX) Interface(n Vertices) sort.Interface { return byX{n} }
func (sortByY) Interface(n Vertices) sort.Interface { return byY{n} }
func (sortByX) Less(c0, c1 Coords) bool { 
	return c0[0] < c1[0] 
}
func (sortByY) Less(c0, c1 Coords) bool { 
	return c0[1] < c1[1] 
}

func mergeSort(output, input Vertices, s sorter, con *common.Concurrentizer) {
	if len(input) < conSizeLimit {
		copy(output, input)
		sort.Sort(s.Interface(output))
		return
	}
	m := (len(input) - 1) / 2 + 1
	o0, o1 := output[0:m], output[m:]
	i0, i1 := input[0:m], input[m:]
	w0 := con.Do(len(o0), func () {
		mergeSort(o0, i0, s, con)
	})
	w1 := con.Do(len(o1), func () {
		mergeSort(o1, i1, s, con)
	})
	w0.Wait()
	w1.Wait()
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
	con.Do(len(output), func () {
		mergeSort(output, input, s, con)
	}).Wait()
	return output
}

func (t *Tree) Build(graph []Vertex) {
	xdim := make(Vertices, len(graph))
	ydim := make(Vertices, len(graph))
	for i, v := range graph {
		xdim[i] = v
		ydim[i] = v
	}
	xdim = concurrentSort(xdim, sortByX{})
	ydim = concurrentSort(ydim, sortByY{})
	log.Println("Sort finished")
	tmp := make(Vertices, len(graph))
	con := common.NewConcurrentizer(conSizeLimit)
	con.Do(len(graph), func() {
		t.root = buildTree(xdim, ydim, tmp, sortByX{}, sortByY{}, con)
	}).Wait()
}

func (t *Tree) FindExact(point Coords) Vertex {
	sorts := []sorter{sortByX{}, sortByY{}}
	v := t.root
	for d := 0; ; d++ {
		s := sorts[d%2]
		if point.Equals(v.Point()) {
			return v
		}
		if s.Less(point, v.Point()) {
			v = v.Left()
		} else {
			v = v.Right()
		}
	}
	return nil
}

// findMidpoint ensures that the midpoint is a true split, i.e., it is
// a lower bound of some point in this dimension.
func findMidpoint(dim Vertices, s sorter) int {
	mid := (len(dim) + 1) / 2 - 1
	for mid > 0 {
		if !s.Less(dim[mid-1].Point(), dim[mid].Point()) {
			mid--
		} else {
			break
		}
	}
	return mid
}

func buildTree(
	thisDim, nextDim, tmpDim Vertices, 
	thisSort, nextSort sorter,
	con *common.Concurrentizer) Vertex {
	if len(thisDim) == 0 {
		return nil
	} 
	// Choose the median point in thisDim.
	mid := findMidpoint(thisDim, thisSort)
	split := thisDim[mid]

	thisLeft, splitLeft := thisDim[0:mid], tmpDim[0:mid]
	thisRight, splitRight := thisDim[mid+1:], tmpDim[mid+1:]

	// Split nextDim into two halves in this plane
	for i, l, r := 0, 0, 0; i < len(thisDim); i++ {
		p := nextDim[i]
		if p == split { 
			continue 
		}
		if thisSort.Less(p.Point(), split.Point()) {
			splitLeft[l] = p
			l++
		} else {
			splitRight[r] = p
			r++
		}
	}
	tmpLeft := nextDim[0:mid]
	tmpRight := nextDim[mid+1:]
	lh := con.Do(len(splitLeft), func () {
		split.SetLeft(buildTree(splitLeft, thisLeft, tmpLeft,
			nextSort, thisSort, con))
	})
	rh := con.Do(len(splitRight), func() {
		split.SetRight(buildTree(splitRight, thisRight, tmpRight,
			nextSort, thisSort, con))
	})
	lh.Wait()
	rh.Wait()
	return split
}
