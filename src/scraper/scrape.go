package scraper

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
	P      Page
	Action string
	Data   []byte
	Err    error
}
