package main

import "log"
import "net/http"
import "io/ioutil"
import "sync"

import "boards"
import "scraper"

const scrapeHeader = "Scraper-Token"

var smutex sync.Mutex
var smap map[string]scraper.Scrape = make(map[string]scraper.Scrape)

// start is the initial URI contacted by the headless browser.
func start(w http.ResponseWriter, r *http.Request) {
        contents, err := ioutil.ReadFile("scraper/scraper.js");
 	if err != nil {
		http.Error(w, "Can't read js", 404)
 	}
	w.Write([]byte("<html><script language=\"javascript\">\n"))
        w.Write(contents); 
	w.Write([]byte("</script></html>\n"))
	log.Print("Starting a scraper...")
}

// scrape is the URI used to retrieve another document to evaluate.
func scrape(w http.ResponseWriter, r *http.Request, 
	ch <-chan scraper.Scrape) {
	item := <- ch
	id := item.Id()
	log.Println("Handing work to a scraper", item)
	w.Header().Add(scrapeHeader, id)
	w.Write(item.Body())
	smutex.Lock()
	smap[id] = item
	smutex.Unlock()
}

// scrapeHandler returns a HTTP handler for scraping items from a
// channel produced by the load board.
func scrapeHandler(ch <-chan scraper.Scrape) func
	(http.ResponseWriter, *http.Request) {
	return func (w http.ResponseWriter, r *http.Request) {
		scrape(w, r, ch)
	}
}

func response(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get(scrapeHeader)
	log.Println("Scraper finished work", id)
	smutex.Lock()
	item, present := smap[id]
	delete(smap, id)
	smutex.Unlock()
	if present {
		body, err := ioutil.ReadAll(r.Body)
		item.Scraped(body, err)
	} else {
		log.Println("Scraper could not find response item", item)
	}
}

// loadBoard produces items for scraping on the channel.
func loadBoard(ch chan<- scraper.Scrape) error {
	tt, err := boards.NewTrulos()
	if err != nil {
		return err
	}
	if err := tt.Init(); err != nil {
		return err
	}
	go tt.Read(ch)
	return nil
}

func main() {
	ch := make(chan scraper.Scrape)
	if err := loadBoard(ch); err != nil {
		log.Fatal("Couldn't initialize load board: ", err)
	}
	http.HandleFunc("/start", start)
	http.HandleFunc("/scrape", scrapeHandler(ch))
	http.HandleFunc("/response", response)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
