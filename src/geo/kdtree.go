package geo

import "runtime"
import "sort"

// Tree is a K-D Tree with K=2
type Tree struct {
	arrays [3][]Node
}

type Node interface {
	Coord() []ScaledRad
}

func NewTree() *Tree {
	return &Tree{}
}

const (
	seqSortLimit = 100000
)

type Nodes []Node

func (n Nodes) Len() int { return len(n) }
func (n Nodes) Swap(i, j int) { n[i], n[j] = n[j], n[i] }

type byX struct { Nodes }
type byY struct { Nodes }

func (bx byX) Less(i, j int) bool { 
	return bx.Nodes[i].Coord()[0] < bx.Nodes[j].Coord()[0] 
}
func (by byY) Less(i, j int) bool { 
	return by.Nodes[i].Coord()[1] < by.Nodes[j].Coord()[1] 
}

type sorter interface {
	Interface(n Nodes) sort.Interface
	Less(n1, n2 Node) bool
}

type sortByX struct {}
type sortByY struct {}

func (sortByX) Interface(n Nodes) sort.Interface { return byX{n} }
func (sortByY) Interface(n Nodes) sort.Interface { return byY{n} }
func (sortByX) Less(n0, n1 Node) bool { return n0.Coord()[0] < n1.Coord()[0] }
func (sortByY) Less(n0, n1 Node) bool { return n0.Coord()[1] < n1.Coord()[1] }

func mergeSort(output, input Nodes, s sorter,
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

func merge(out, in0, in1 Nodes, s sorter) {
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

func concurrentSort(input Nodes, s sorter) Nodes {
	procs := runtime.GOMAXPROCS(0)
	running := make(chan bool, procs)
	output := make(Nodes, len(input))
	done := make(chan bool, 1)
	mergeSort(output, input, s, running, done)
	<- done
	return output
}

func (t *Tree) Build(nodes []Node) {
	t.arrays[0] = make(Nodes, len(nodes))
	t.arrays[1] = make(Nodes, len(nodes))
	for i, node := range nodes {
		t.arrays[0][i] = node
		t.arrays[1][i] = node
	}
	t.arrays[0] = concurrentSort(t.arrays[0], sortByX{})
	t.arrays[1] = concurrentSort(t.arrays[1], sortByY{})
}
