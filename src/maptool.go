package main

import "flag"
import "fmt"
import "io"
import "log"
import "os"
import "runtime"
import "geo"

import "maps"

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
	"living_street": true,
	"residential": true,
	"unclassified": true,
	"service": true,
	"road": true,
}

type nodeId uint32
type mapId int64

type mapCount struct {
	id nodeId
	ec uint32
}

type mapData1 struct {
	// The set of node ID we wish to keep during the following
	// scan and renumberings into a (32-bit dense ID number,
	// 32-bit count of outgoing edges)
	mapIds map[mapId]mapCount
	nextNodeId nodeId
	totalEdges uint32
}

type node struct {
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
			if _, has := highwayTypes[a.Value]; has {
				return true
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

func (md *mapData2) mapPass2(bd *maps.BlockData, md1 *mapData1) {
	
}

func readInput() io.Reader {
	file, err := os.Open(*input)
	if err != nil {
		log.Fatalln("Could not open file:", *input, ":", err)
	}
	log.Println("Reading", *input)
	return file
}

func printMem() {
	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)
	log.Println("Total memory allocated:", ms.Alloc)
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
	for _, mc := range md1.mapIds {
		np := &md2.nodes[mc.id]
		np.neighbors = md2.edges[ei:ei+int(mc.ec)]
		ei += int(mc.ec)
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
	osm := maps.NewMap()

	md1 := newMapData1()
	if err := osm.ReadMap(readInput(), func (bd *maps.BlockData) {
		md1.mapPass1(bd)
	}); err != nil {
		log.Fatalln("Error reading map:", *input, ":", err)
	}
	printMem()
	log.Println("Using", md1.nextNodeId, "nodes, have", 
		md1.totalEdges, "edges")

	md2 := newMapData2(md1)
	if err := osm.ReadMap(readInput(), func (bd *maps.BlockData) {
		md2.mapPass2(bd, md1)
	}); err != nil {
		log.Fatalln("Error reading map:", *input, ":", err)
	}
	md1 = nil
	printMem()
}
