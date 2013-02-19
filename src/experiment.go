package main

import "log"
import "net/http"
import "io/ioutil"

//import "net/http/httputil"
import "sync"

import "boards"
import "scraper"

const (
	scrapeHeader = "Scraper-Token"
	actionHeader = "Scraper-Action"
)

type scrapeState struct {
	page scraper.Page

	// Index of the current page.Action() (-1 for the initial action)
	aid int
}

var smutex sync.Mutex
var smap map[string]*scrapeState = make(map[string]*scrapeState)

// start is the initial URI contacted by the headless browser.
func start(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("scraper/scraper.js")
	if err != nil {
		http.Error(w, "Can't read js", 404)
	}
	w.Write([]byte("<html><script type=\"text/javascript\">\n"))
	w.Write(contents)
	w.Write([]byte("</script></html>\n"))
	log.Print("Starting a scraper...")
}

func test(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("boards/trulos.html")
	if err != nil {
		http.Error(w, "Can't read HTML", 404)
	}
	w.Write(contents)
}

// scrape is the URI used to retrieve another document to evaluate.
func scrape(w http.ResponseWriter, r *http.Request,
	pages <-chan scraper.Page) {
	page := <-pages
	id := page.Id()
	log.Println("Handing work to a scraper", page)
	w.Header().Add(scrapeHeader, id)
	w.Write(page.Body())
	smutex.Lock()
	smap[id] = &scrapeState{page, -1}
	smutex.Unlock()
}

// scrapeHandler returns a HTTP handler for scraping items from a
// channel produced by the load board.
func scrapeHandler(pages <-chan scraper.Page) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		scrape(w, r, pages)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	//dump, _ := httputil.DumpRequest(r, true)
	log.Println("Default handler:", r.Method, r.URL)
	w.WriteHeader(http.StatusNotFound)
}

func response(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get(scrapeHeader)
	log.Println("Scraper finished work", id)
	smutex.Lock()
	state, present := smap[id]
	smutex.Unlock()
	if !present {
		log.Println("Scraper could not find response item", id)
		return
	}
	page := state.page
	body, err := ioutil.ReadAll(r.Body)
	actions := page.Actions()
	act := ""
	if state.aid >= 0 {
		act = actions[state.aid]
	}
	page.Channel() <- &scraper.Result{page, act, body, err}

	state.aid++
	if state.aid < len(actions) {
		next := string(actions[state.aid])
		log.Println("Sending next action", next)
		w.Header().Add(actionHeader, next)
	} else {
		smutex.Lock()
		delete(smap, id)
		smutex.Unlock()
	}
}

// loadBoard produces items for scraping on the channel.
func loadBoard(ch chan<- scraper.Page) error {
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
	ch := make(chan scraper.Page)
	if err := loadBoard(ch); err != nil {
		log.Fatal("Couldn't initialize load board: ", err)
	}
	http.HandleFunc("/test", test)
	http.HandleFunc("/start", start)
	http.HandleFunc("/scrape", scrapeHandler(ch))
	http.HandleFunc("/response", response)
	http.HandleFunc("/", handle)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
