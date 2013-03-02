// Interface for reading load information from any load board.

package boards

import "time"

import "scraper"

type LoadBoard interface {
	Init() error
	Read(chan<- scraper.Page)
}

type Load struct {
	PickupDate  time.Time
	OriginCity  string
	OriginState string
	DestCity    string
	DestState   string
	LoadType    string   // "Full" or "Partial" or ?
	Length      int
	Weight      int
	Equipment   string
	Price       float64
	Stops       int
	Phone       string
}
