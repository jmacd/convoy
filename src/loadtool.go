package main

import "database/sql"
import "errors"
import "flag"
import "fmt"
import "log"
import "runtime"
import "time"

import "boards"
import "data"
import "scraper"

type LoadSet struct {
	data.ConvoyData

	// Number of days in this load set
	days int

	// Loads appear on a day numbered from 0.  This maps scrape ID
	// to day number.
	dayZero time.Time
	scrapeToDay map[int64]int
	dayToScrape []int64
	dayToDate []time.Time

	// Map of day number to loads, grouped and counted, de-duped, etc.
	loads []map[boards.Load]int

	// Some statistics
	dups, sameday, pastdate, repeat, ltrepeat, gtrepeat, total int
}

func (l *LoadSet) daysFromZero(t time.Time) int {
	diff := t.Sub(l.dayZero)
	// Note: 16 / 24 is > 0.5 to account for daylight savings time
	// changes, to round up when the number of hours is not a
	// multiple of 24.
	return int((diff.Hours() + 16.0) / 24.0)
}

func (ls *LoadSet) readScrapes() error {
	var scrapes []scraper.Scrape
	if err := ls.ConvoyData.ForAllScrapes(func (scrape scraper.Scrape) error {
		if scrape.StartTime.IsZero() || scrape.FinishTime.IsZero() {
			return errors.New(
				fmt.Sprint("Incomplete scrape id: ", scrape.ScrapeId))
		}
		scrapes = append(scrapes, scrape)
		return nil
	}); err != nil {
		return err
	}

	for _, s := range scrapes {
		if ls.dayZero.IsZero() {
			ls.dayZero = s.Date()
			continue
		}
		if ls.dayZero.After(s.Date()) {
			ls.dayZero = s.Date()
		}
	}
	maxDay := 0
	for _, s := range scrapes {
		dayOffset := ls.daysFromZero(s.Date())
		ls.scrapeToDay[s.ScrapeId] = dayOffset
		if dayOffset > maxDay {
			maxDay = dayOffset
		}
	}
	ls.days = maxDay+1
	ls.dayToScrape = make([]int64, ls.days)
	ls.dayToDate = make([]time.Time, ls.days)
	for id, day := range ls.scrapeToDay {
		if ls.dayToScrape[day] != 0 {
			return errors.New(fmt.Sprint("Duplicate scrape date: ", id))
		}
		ls.dayToScrape[day] = id
		ls.dayToDate[day] = ls.dayZero.AddDate(0, 0, day)
	}
	log.Printf("%d scrapes; %d days", len(scrapes), ls.days)
	return nil
}

func (ls *LoadSet) removeRepost(tday int, tload, yload boards.Load) {
	tcnt, has := ls.loads[tday][tload]
	if !has {
		return
	}
	yday := tday - 1
	if ycnt, has := ls.loads[yday][yload]; has {
		if ycnt == tcnt {
			ls.repeat++
			delete(ls.loads[tday], tload)
		} else if ycnt > tcnt {
			ls.ltrepeat++
			delete(ls.loads[tday], tload)
		} else if ycnt < tcnt {
			ls.loads[tday][tload] = tcnt - ycnt
			ls.gtrepeat++
		}
	}
}

func (ls *LoadSet) readLoads() error {
	ls.loads = make([]map[boards.Load]int, ls.days)
	for i, _ := range ls.loads {
		ls.loads[i] = make(map[boards.Load]int)
	}
	if err := ls.ConvoyData.ForAllLoads(func (load boards.Load) error {
		day := ls.scrapeToDay[load.ScrapeId]
		date := ls.dayToDate[day]
		if date.Equal(load.PickupDate) {
			ls.sameday++
		} else if date.After(load.PickupDate) {
			ls.pastdate++
			return nil
		}
		load.ScrapeId = 0
		if cnt, has := ls.loads[day][load]; has {
			ls.loads[day][load] = cnt + 1
			ls.dups++
		} else {
			ls.loads[day][load] = 1
		}
		return nil
	}); err != nil {
		return err
	}

	// Remove next-day re-posts (assume unsatisfied), duplicates, etc.
	for day := ls.days-1; day > 0; day-- {
		for load, _ := range ls.loads[day] {
			ls.removeRepost(day, load, load)
			if ls.dayToDate[day].Equal(load.PickupDate) {
				yload := load
				yload.PickupDate = ls.dayToDate[day-1]
				ls.removeRepost(day, load, yload)
			}
		}
	}
	for _, loadmap := range ls.loads {
		for _, cnt := range loadmap {
			ls.total += cnt
		}
	}

	log.Println("Duplicates", ls.dups, 
		"Sameday", ls.sameday,
		"Pastdate", ls.pastdate,
		"Repeat", ls.repeat,
		"<Repeat", ls.ltrepeat,
		">Repeat", ls.gtrepeat,
		"Total", ls.total)
	return nil
}

func NewLoadSet(db *sql.DB) (*LoadSet, error) {
	ls := &LoadSet{}
	cd, err := data.NewConvoyData(db)
	if err != nil {
		return nil, err
	}
	ls.ConvoyData = *cd
	ls.scrapeToDay = make(map[int64]int)
	
	if err = ls.readScrapes(); err != nil {
		return nil, err
	}

	return ls, ls.readLoads()
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
		log.Fatalln("Could not open database", err)
	}
	defer db.Close()

	ls, err := NewLoadSet(db)
	if err != nil {
		log.Fatalln("Could not read loads", err)
	}
	_ = ls
}
