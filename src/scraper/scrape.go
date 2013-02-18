package scraper;

type Scrape interface {
	Id() string
	Actions() [][]byte
	Body() []byte
	Scraped([]byte, error)
}