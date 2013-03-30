package main

import "flag"
import "fmt"
import "io"
import "log"
import "os"
import "runtime"

import "common"
import "data"
import "geo"
import "maps"
//import "graph"

var input = flag.String("input", "", "OSM PBF formatted file")

var highwayTypes = map[string]bool {
	"motorway": true,
	"motorway_link": true,
	"trunk": true,
	"trunk_link": true,
	"primary": true,
	"primary_link": true,
	"secondary": true,
	"secondary_link": true,
	"tertiary": true,
	"tertiary_link": true,
	"living_street": false,
	"residential": false,
	"unclassified": false,
	"service": false,
	"road": false,
}

type nodeId uint32
type mapId int64

type mapCount struct {
	id nodeId
	ec uint32
}

type mapData1 struct {
	// The set of node IDs we wish to keep during the following
	// scan and renumberings into a (32-bit dense ID number,
	// 32-bit count of outgoing edges)
	mapIds map[mapId]mapCount
	nextNodeId nodeId
	totalEdges uint32
}

type node struct {
	id nodeId
	position [3]geo.EarthLoc
	treeLeft, treeRight nodeId
	neighbors []nodeId
}

type mapData2 struct {
	nodes []node
	edges []nodeId
}

func keepWay(way *maps.Way) bool {
	for _, a := range way.Attrs {
		if a.Key == "highway" {
			if yes, has := highwayTypes[a.Value]; has {
				return yes
			}
		}
	}
	return false
}

func (md *mapData1) mapPass1(bd *maps.BlockData) {
	for w := 0; w < len(bd.Ways); w++ {
		way := &bd.Ways[w]
		if !keepWay(way) {
			continue
		}
		if len(way.Refs) < 2 {
			continue
		}
		md.totalEdges += uint32(len(way.Refs)) - 1
		for e, ref := range way.Refs {
			var edges uint32
			if e == 0 || e == (len(way.Refs) - 1) {
				edges = 1
			} else {
				edges = 2
			}

			value, has := md.mapIds[mapId(ref)]
			if has {
				value.ec += edges
				md.mapIds[mapId(ref)] = value
				continue
			}

			md.mapIds[mapId(ref)] = mapCount{md.nextNodeId, edges}
			md.nextNodeId++
		}
	}
}

func (md *mapData2) addEdge(n0 *node, v1 nodeId) {
	for i, neighbor := range n0.neighbors {
		if neighbor != 0 {
			continue
		}
		n0.neighbors[i] = v1
	}
}

func (md *mapData2) addEdges(v0, v1 nodeId) {
	md.addEdge(&md.nodes[v0], v1)
	md.addEdge(&md.nodes[v1], v0)
}

func (md *mapData2) mapPass2(bd *maps.BlockData, md1 *mapData1) {
	for n := 0; n < len(bd.Nodes); n++ {
		mapnode := &bd.Nodes[n]
		mc, has := md1.mapIds[mapId(mapnode.Id)]
		if !has {
			continue
		}
		mn := &md.nodes[mc.id]
		mn.id = mc.id
		geo.LatLongDegreesToCoords(
			geo.SphereCoords{mapnode.Lat, mapnode.Long}, mn.position[:])
	}
	for w := 0; w < len(bd.Ways); w++ {
		way := &bd.Ways[w]
		if !keepWay(way) {
			continue
		}
		if len(way.Refs) < 2 {
			continue
		}
		for e := 1; e < len(way.Refs); e++ {
			mc0, has0 := md1.mapIds[mapId(way.Refs[e-1])]
			mc1, has1 := md1.mapIds[mapId(way.Refs[e])]
			if !has0 || !has1 {
				panic("Corrupted mapIds?")
			}
			md.addEdges(mc0.id, mc1.id)
		}
	}	
}

func readInput() io.Reader {
	file, err := os.Open(*input)
	if err != nil {
		log.Fatalln("Could not open file:", *input, ":", err)
	}
	log.Println("Reading", *input)
	return file
}

func newMapData1() *mapData1 {
	return &mapData1{
		mapIds: make(map[mapId]mapCount),
		nextNodeId: nodeId(1),
		totalEdges: 0,
	}
}

func newMapData2(md1 *mapData1) *mapData2 {
	md2 := &mapData2{
		make([]node, md1.nextNodeId),
		make([]nodeId, md1.totalEdges * 2),
	}
	ei := 0
	c := 0
	for _, mc := range md1.mapIds {
		np := &md2.nodes[mc.id]
		np.neighbors = md2.edges[ei:ei+int(mc.ec)]
		ei += int(mc.ec)
		c++
	}
	if ei != len(md2.edges) {
		panic(fmt.Sprintln("Incorrect edge count", ei, len(md2.edges)))
	}
	return md2
}

func main() {
	flag.Parse()
	argv := flag.Args()
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(argv) != 0 {
		log.Fatalln("Extra args:", argv)
	}
	db, err := data.OpenDb()
	if err != nil {
		log.Fatal("Could not open database", err)
	}
	defer db.Close()

	osm := maps.NewMap()

	md1 := newMapData1()
	if err := osm.ReadMap(readInput(), func (bd *maps.BlockData) {
		md1.mapPass1(bd)
	}); err != nil {
		log.Fatalln("Error reading map:", *input, ":", err)
	}
	common.PrintMem()
	log.Println("Using", md1.nextNodeId, "nodes, have", 
		md1.totalEdges, "edges")

	md2 := newMapData2(md1)
	if err := osm.ReadMap(readInput(), func (bd *maps.BlockData) {
		md2.mapPass2(bd, md1)
	}); err != nil {
		log.Fatalln("Error reading map:", *input, ":", err)
	}
	md1 = nil
	common.PrintMem()
	
	// Sanity check: should have filled-in all edges
	for _, e := range md2.edges {
		if e == nodeId(0) {
			panic("Did not fill-in all edges")
		}
	}

	tree := geo.NewTree(md2)
	tree.Build()
	log.Println("Built geospatial tree")
	common.PrintMem()

	// TODO(jmacd), and then...
	// if err := printCityDistances(db, tree); err != nil {
	// 	log.Println("PrintCityDistances:", err)
	// }
}

func (n *node) Point() geo.Coords {
	return n.position[:]
}

func (n *node) Left(g geo.Graph) geo.Vertex {
	if n.treeLeft == 0 {
		return nil 
	}
	return &g.(*mapData2).nodes[n.treeLeft]
}

func (n *node) Right(g geo.Graph) geo.Vertex {
	if n.treeRight == 0 {
		return nil
	}
	return &g.(*mapData2).nodes[n.treeRight]
}

func (n *node) SetLeft(g geo.Graph, v geo.Vertex) {
	if v != nil {
		n.treeLeft = v.(*node).id
	}
}

func (n *node) SetRight(g geo.Graph, v geo.Vertex) {
	if v != nil {
		n.treeRight = v.(*node).id
	}
}

func (n *node) String() string {
	return fmt.Sprintf("(%v:%v,%v,%v)", 
		n.id, n.position[0], n.position[1], n.position[2])
}

func (md *mapData2) Count() int {
	return len(md.nodes) - 1
}

func (md *mapData2) Node(i int) geo.Vertex {
	return &md.nodes[i + 1]
}
