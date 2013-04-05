package main

import "flag"
//import "fmt"
import "log"
import "runtime"

import "boards"
import "data"
import "scraper"

// Loads appear on a day numbered from 0.  This maps scrape ID to day
// number.
var scrapeToDay map[int64]int

// Map of day number f
var byScrape map[int][]boards.Load

func main() {
	flag.Parse()
	argv := flag.Args()
	runtime.GOMAXPROCS(runtime.NumCPU())
	if len(argv) != 0 {
		log.Fatalln("Extra args:", argv)
	}
	db, err := data.OpenDb()
	if err != nil {
		log.Fatalln("Could not open database", err)
	}
	defer db.Close()

	cd, err := data.NewConvoyData(db)
	if err != nil {
		log.Fatalln("Could not prepare database", err)
	}

	err = cd.ForAllScrapes(func (scrape scraper.Scrape) error {

		return nil
	})

	err = cd.ForAllLoads(func (load boards.Load) error {

		return nil
	})
	if err != nil {
		log.Fatalln("Load scan failed", err)
	}
}
