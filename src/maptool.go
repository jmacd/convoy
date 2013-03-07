package main

import "flag"
import "log"
import "os"

import "maps"

var input = flag.String("input", "", "OSM PBF formatted file")

func main() {
	flag.Parse()
	argv := flag.Args()
	if len(argv) != 0 {
		log.Fatalln("Extra args:", argv)
	}
	file, err := os.Open(*input)
	if err != nil {
		log.Fatalln("Could not open file:", *input, ":", err)
	}
	if err = maps.ReadMap(file); err != nil {
		log.Fatalln("Error reading map:", *input, ":", err)
	}
}
