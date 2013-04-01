package graph

import "container/heap"

type NodeId uint32
type heapPos int32

const (
	ZeroNodeId NodeId = 0
	FirstNodeId NodeId = 1

	unknown heapPos = -1
	settled heapPos = -2
)

type Graph interface {
	Count() int
	Neighbors(id NodeId) []NodeId
	Distance(from, to NodeId) float32
}

type qpos struct {
	// Id of this node in the graph.
        id        NodeId
	
	// Position of this node in the queue.
        index     heapPos

	// Cost of the shortest path to this node.
        cost      float32

	// Tentative or settled parent node.
        parent    NodeId
}

type queue struct {
	heap []*qpos
	data []qpos
}	

type dijkstra struct {
	g  Graph
        q  [2]queue  // [0] = forward, [1] = reverse
}

func ShortestPath(g Graph, start, end NodeId) []NodeId {
	if start == end {
		return []NodeId{start}
	}

        d := newDijkstra(g)
        d.q[0].visit(start, 0.0, ZeroNodeId)
        d.q[1].visit(end, 0.0, ZeroNodeId)

	// The midpoint
	var mid NodeId

	// Search, during which nodes are queued, settled, or unknown.
	dir := 1
	rounds := 0
	for {
		odir := dir
		dir = 1 - dir
		q := &d.q[dir]
		oq := &d.q[odir]

		if q.empty() {
			return nil
		}

		p := q.next()
		op := &oq.data[p.id]

		if op.index == settled {
			mid = p.id
			break
		}
		p.index = settled
		neighbors := d.g.Neighbors(p.id)
		for _, n := range neighbors {
			q.visit(n, p.cost + g.Distance(p.id, n), p.id)
		}
		rounds++
	}
	countParents := func (child, root NodeId, q *queue) int {
		count := 0
		for child != root {
			count++
			child = q.data[child].parent
		}
		return count
	}
	fcount := countParents(mid, start, &d.q[0])
	rcount := countParents(mid, end, &d.q[1])
	path := make([]NodeId, fcount + rcount + 1)

	fillPath := func (child, root NodeId, idx, incr int, q *queue) {
		for child != root {
			child = q.data[child].parent
			path[idx] = child
			idx += incr
		}
	}
	fillPath(mid, start, fcount-1, -1, &d.q[0])
	path[fcount] = mid
	fillPath(mid, end, fcount+1, +1, &d.q[1])
	return path
}

func newDijkstra(g Graph) *dijkstra {
	nodes := int(g.Count()+1)
        d := &dijkstra{g: g}
        d.q[0].data = make([]qpos, nodes)
        d.q[1].data = make([]qpos, nodes)
	for i := 1; i < nodes; i++ {
		d.q[0].data[i].id = NodeId(i)
		d.q[1].data[i].id = NodeId(i)
		d.q[0].data[i].index = unknown
		d.q[1].data[i].index = unknown
	}
        return d
}

func (q *queue) visit(id NodeId, cost float32, parent NodeId) {
        p := &q.data[id]
	if p.index == settled {
 		return
	}
        if p.index == unknown {
                p.parent = parent
                p.cost = cost
		heap.Push(q, p)
		return
        }
	if cost >= p.cost {
		return
	}
	heap.Remove(q, int(p.index))
	p.cost = cost
	p.parent = parent
	heap.Push(q, p)
}

func (q *queue) empty() bool {
        return len(q.heap) == 0
}

func (q *queue) next() *qpos {
        return heap.Pop(q).(*qpos)
}

func (q *queue) Len() int {
        return len(q.heap)
}

func (q *queue) Less(i, j int) bool {
        return q.heap[i].cost < q.heap[j].cost
}

func (q *queue) Swap(i, j int) {
        q.heap[i], q.heap[j] = q.heap[j], q.heap[i]
        q.heap[i].index = heapPos(i)
        q.heap[j].index = heapPos(j)
}

func (q *queue) Push(x interface{}) {
        p := x.(*qpos)
        p.index = heapPos(len(q.heap))
        q.heap = append(q.heap, p)
}

func (q *queue) Pop() interface{} {
        n := len(q.heap)
        x := q.heap[n-1]
        q.heap = q.heap[:n-1]
        x.index = unknown
        return x
}
