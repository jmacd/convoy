// Interface for reading load information from any load board.

package boards

import "time"

import "scraper"

type LoadBoard interface {
	Init() error
	Read(chan<- scraper.Scrape)
}

type Load struct {
	Date time.Time
	OriginCity string
	OriginState string
	DestCity string
	DestState string
	Load string
	Length int
	Weight int
	Equipment string
	Price float64
	Stops int
	Phone string
}
