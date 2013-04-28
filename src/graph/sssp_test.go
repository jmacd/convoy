package graph

import "testing"

func (g *graph) check(t *testing.T, n0, n1 NodeId, expect []NodeId) {
	s := ShortestPath(g, n0, n1)
	if len(s) != len(expect) {
		t.Errorf("Incorrect sssp length got %v want %v", s, expect)
		return
	}
	for i := 0; i < len(s); i++ {
		if s[i] != expect[i] {
			t.Errorf("Incorrect sssp entry[%v] got %v want %v", i, s, expect)
			return
		}
	}
}

func TestSssp(t *testing.T) {
	g := newGraph()
	n0 := g.addNode()

	g.check(t, n0, n0, []NodeId{n0})
	
	n1 := g.addNode()
	g.check(t, n0, n1, []NodeId{})

	g.addEdge(n0, n1, 1.0)
	g.check(t, n0, n1, []NodeId{n0, n1})

	n2 := g.addNode()
	g.check(t, n0, n2, []NodeId{})
	g.check(t, n2, n1, []NodeId{})

	g.addEdge(n1, n2, 1.0)
	g.check(t, n0, n2, []NodeId{n0, n1, n2})
	g.check(t, n2, n0, []NodeId{n2, n1, n0})

	n3 := g.addNode()
	g.addEdge(n0, n3, 0.5)
	g.addEdge(n3, n2, 0.5)
	g.check(t, n0, n2, []NodeId{n0, n3, n2})

	n4 := g.addNode()
	g.addEdge(n3, n4, 5.0)
	g.addEdge(n2, n4, 1.0)
	g.check(t, n0, n4, []NodeId{n0, n3, n2, n4})
}