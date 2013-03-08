package main

import "flag"
import "log"
import "os"
import "runtime"

import "maps"

var input = flag.String("input", "", "OSM PBF formatted file")

func main() {
	flag.Parse()
	argv := flag.Args()
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(argv) != 0 {
		log.Fatalln("Extra args:", argv)
	}
	file, err := os.Open(*input)
	if err != nil {
		log.Fatalln("Could not open file:", *input, ":", err)
	}
	osm := maps.NewMap()
	if err = osm.ReadMap(file); err != nil {
		log.Fatalln("Error reading map:", *input, ":", err)
	}
}
