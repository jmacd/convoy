package graph;

import "testing"

func TestCondense1(t *testing.T) {
	g := newGraph()
	n0 := g.addNode()
	n1 := g.addNode()
	n2 := g.addNode()
	n3 := g.addNode()
	n4 := g.addNode()
	n5 := g.addNode()
	g.addEdge(n0, n1, 100.0)
	g.addEdge(n1, n2, 100.0)
	g.addEdge(n2, n3, 100.0)
	g.addEdge(n3, n4, 100.0)
	g.addEdge(n4, n5, 100.0)
	
	k1 := func (n NodeId) bool {
		return n == n0 || n == n5
	}
	cedges := Condense(g, k1)

	if len(cedges) != 1 {
		t.Errorf("Expected 1 edges, got %v", cedges)
		return
	}
	if int(cedges[0].weight) != 500 {
		t.Errorf("Expected cost 500, got %.3f", cedges[0].weight)
		return
	}

	n6 := g.addNode()
	g.addEdge(n3, n6, 200)
	
}
