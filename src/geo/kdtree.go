package geo

import "runtime"
import "sort"
import "log"

const (
	seqSortLimit = 100000
)

type Vertices []Vertex

// Tree is a K-D Tree with K=2
type Tree struct {
	root Vertex
}

type Vertex interface {
	Coord() []ScaledRad
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
	Less(n1, n2 Vertex) bool
}

type sortByX struct {}
type sortByY struct {}

type byX struct { Vertices }
type byY struct { Vertices }

func (bx byX) Less(i, j int) bool { 
	return bx.Vertices[i].Coord()[0] < bx.Vertices[j].Coord()[0] 
}
func (by byY) Less(i, j int) bool { 
	return by.Vertices[i].Coord()[1] < by.Vertices[j].Coord()[1] 
}

func (sortByX) Interface(n Vertices) sort.Interface { return byX{n} }
func (sortByY) Interface(n Vertices) sort.Interface { return byY{n} }
func (sortByX) Less(n0, n1 Vertex) bool { return n0.Coord()[0] < n1.Coord()[0] }
func (sortByY) Less(n0, n1 Vertex) bool { return n0.Coord()[1] < n1.Coord()[1] }

func mergeSort(output, input Vertices, s sorter,
	running chan bool, done chan<- bool) {
	if len(input) < seqSortLimit {
		copy(output, input)
		sort.Sort(s.Interface(output))
		done <- true
		return
	}
	child := make(chan bool, 2)
	m := (len(input) - 1) / 2 + 1
	o0, o1 := output[0:m], output[m:]
	i0, i1 := input[0:m], input[m:]
	go mergeSort(o0, i0, s, running, child)
	go mergeSort(o1, i1, s, running, child)
	<-child
	<-child
	running <- true
	copy(input, output)
	merge(output, i0, i1, s)
	<-running
	done <- true
}

func merge(out, in0, in1 Vertices, s sorter) {
	i, j, k := 0, 0, 0
	for ; i < len(in0) && j < len(in1); k++ {
		if s.Less(in0[i], in1[j]) {
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
	procs := runtime.GOMAXPROCS(0)
	running := make(chan bool, procs)
	output := make(Vertices, len(input))
	done := make(chan bool, 1)
	mergeSort(output, input, s, running, done)
	<- done
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
	t.root = t.buildTree(xdim, ydim, tmp, sortByX{}, sortByY{})
}

// findMidpoint ensures that the midpoint is a true split, i.e., it is
// a lower bound of some point in this dimension.
func findMidpoint(dim Vertices, s sorter) int {
	mid := (len(dim) + 1) / 2 - 1
	for mid > 0 {
		if !s.Less(dim[mid-1], dim[mid]) {
			mid--
		} else {
			break
		}
	}
	return mid
}

func (t *Tree) buildTree(thisDim, nextDim, tmpDim Vertices, 
	thisSort, nextSort sorter) Vertex {
	if len(thisDim) == 0 {
		return nil
	}
	// Choose the median point in thisDim TODO(jmacd) pay
	// attention to finding a true lower bound in case of
	// duplicate values
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
		if thisSort.Less(p, split) {
			splitLeft[l] = p
			l++
		} else {
			splitRight[r] = p
			r++
		}
	}
	tmpLeft := nextDim[0:mid]
	tmpRight := nextDim[mid+1:]
	leftChild := t.buildTree(splitLeft, thisLeft, tmpLeft, 
		nextSort, thisSort)
	rightChild := t.buildTree(splitRight, thisRight, tmpRight, 
		nextSort, thisSort)
	if leftChild != nil {
		split.SetLeft(leftChild)
	}
	if rightChild != nil {
		split.SetRight(rightChild)
	}
	return split
}
