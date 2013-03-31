package graph

type Vertex interface {
//	Neighbors() []int
}

type Graph interface {
//	Count() int
//	Node(int) Vertex
}

func ShortestPath(g Graph, start, end Vertex) []Vertex {
	return []Vertex{}
}

//        d := newDijkstra(graph)

//         d.visit(start, 0, nil)
//         for !d.empty() {
//                 p := d.next()
//                 if g.Node(p.v) == g.Node(end) {
//                         break
//                 }
//                 for _, v := range g.Neighbors(p.v) {
//                         d.visit(v, p.depth+1, p)
//                 }
//         }

//         p := d.pos(end)
//         if p.depth == 0 {
//                 // unvisited - no path
//                 return nil
//         }
//         path := make([]Vertex, p.depth)
//         for ; p != nil; p = p.parent {
//                 path[p.depth-1] = p.v
//         }
//         return path
// }

// // A dpos is a position in the Dijkstra traversal.
// type dpos struct {
//         cost      float64
//         heapIndex int
//         v         Vertex
//         parent    *dpos
// }

// // A dijkstra is the Dijkstra traversal's work state.
// // It contains the heap queue and per-vertex information.
// type dijkstra struct {
//         g    Graph
//         q    []*dpos
//         byID []dpos
// }

// func newDijkstra(g Graph) *dijkstra {
//         d := &dijkstra{g: g}
//         d.byID = make([]dpos, g.NumVertex())
//         return d
// }

// func (d *dijkstra) pos(v Vertex) *dpos {
//         p := &d.byID[d.g.Node(v)]
//         p.v = v // in case this is the first time we've seen it
//         return p
// }

// func (d *dijkstra) visit(v Vertex, depth int, parent *dpos) {
//         p := d.pos(v)
//         if p.depth == 0 {
//                 p.parent = parent
//                 p.depth = depth
//                 heap.Push(d, p)
//         }
// }

// func (d *dijkstra) empty() bool {
//         return len(d.q) == 0
// }

// func (d *dijkstra) next() *dpos {
//         return heap.Pop(d).(*dpos)
// }

// // Implementation of heap.Interface
// func (d *dijkstra) Len() int {
//         return len(d.q)
// }

// func (d *dijkstra) Less(i, j int) bool {
//         return d.q[i].depth < d.q[j].depth
// }

// func (d *dijkstra) Swap(i, j int) {
//         d.q[i], d.q[j] = d.q[j], d.q[i]
//         d.q[i].heapIndex = i
//         d.q[j].heapIndex = j
// }

// func (d *dijkstra) Push(x interface{}) {
//         p := x.(*dpos)
//         p.heapIndex = len(d.q)
//         d.q = append(d.q, p)
// }

// func (d *dijkstra) Pop() interface{} {
//         n := len(d.q)
//         x := d.q[n-1]
//         d.q = d.q[:n-1]
//         x.heapIndex = -1
//         return x
// }
