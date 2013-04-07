package scraper

import "time"

var timeZone *time.Location

func init() {
	timeZone, _ = time.LoadLocation("UTC")
}

// Page describes a page loading & scraping operation.
type Page interface {
	// A list of javascript actions to take after the initial load.
	Actions() []string
	// The body of the page to load.
	Body() []byte
	// A unique identifier for this page scrape.
	Id() string
	// Completion channel
	Channel() chan<- *Result
}

// Result describes the result of a page scrape.  Action is empty
// for the initial page load, subsequently contains the value of
// any Actions() returned by the Page.
type Result struct {
	Action string
	Data   []byte
	Err    error
}

// A single scrape.
type Scrape struct {
	ScrapeId   int64
	StartTime  time.Time
	FinishTime time.Time
}

// Returns the closest average date of start/finish times.
func (s *Scrape) Date() time.Time {
	su := s.StartTime.Unix()
	fu := s.FinishTime.Unix()
	avg := (su + fu) / 2
	half := time.Unix(avg + (24 * 60 * 60) / 2, 0)
	y, m, d := half.Date()
	before := time.Date(y, m, d, 0, 0, 0, 0, timeZone)
	after := time.Date(y, m, d + 1, 0, 0, 0, 0, timeZone)
	if avg - before.Unix() <= after.Unix() - avg {
		return before
	}
	return after
}
