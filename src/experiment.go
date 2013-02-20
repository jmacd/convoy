package main

import "io/ioutil"
import "log"
import "net/http"
import "net/http/httputil"
import "net/url"
import "regexp"
import "sync"
import "time"

import "boards"
import "scraper"

const (
	scrapeToken    = "Scraper-Token"
	scrapeAction    = "Scraper-Action"
	scrapeJsFile    = "scraper/scraper.js"
	validPathRegexp = `^/(Trulos|Truck).*`
)

type scrapeState struct {
	page scraper.Page
	aid  int // Index of the current page.Action() or -1
}

var (
	reverseProxy *httputil.ReverseProxy
	validPathRe  *regexp.Regexp
	scrapeScript []byte
	smutex       sync.Mutex
	smap         map[string]*scrapeState = make(map[string]*scrapeState)
)

func init() {
	contents, err := ioutil.ReadFile(scrapeJsFile)
	if err != nil {
		log.Print("Can't read " + scrapeJsFile)
	}
	scrapeScript = append(scrapeScript, []byte("<script type=\"text/javascript\">\n")...)
	scrapeScript = append(scrapeScript, contents...)
	scrapeScript = append(scrapeScript, []byte("</script>\n")...)

	reverseProxy = &httputil.ReverseProxy{
		proxyFunction,
		&http.Transport{
			Proxy:              removeForwardedForProxy,
			DisableCompression: true},
		time.Duration(0)}

	validPathRe = regexp.MustCompile(validPathRegexp)
}

func removeForwardedForProxy(r *http.Request) (*url.URL, error) {
	// Let's remove Referer (this is added after the ReverseProxy's Proxy
	// function is called).
	r.Header.Del("X-Forwarded-For")
	return http.ProxyFromEnvironment(r)
}

func proxyFunction(r *http.Request) {
	// The default handler rejects non-valid requests, assume all
	// others go to the site itself.
	r.URL.Scheme = "http"
	r.URL.Host = "www.trulos.com"
	r.Host = "www.trulos.com"
	// Remove Referer, If-Modified-Since.
	r.Header.Del("Referer")
	r.Header.Del("If-Modified-Since")
	if r.Method == "POST" {
		r.Header.Del("Origin")
		r.Header.Add("Origin", "http://www.trulos.com")

		// Note: The form action="" field specifies an unqualified
		// path, which produces the scraper's path-directory,
		// incorrectly.
		r.URL.Path = "/Trulos/Post-Truck-Loads/Truck-Load-Board.aspx"
	}
}

// Request is the URI used to attach a scraper.
func scrape(w http.ResponseWriter, r *http.Request,
	pages <-chan scraper.Page) {
	page := <-pages
	id := page.Id()
	log.Println("Handing work to a scraper", page)
	w.Header().Add(scrapeToken, id)
	w.Write(page.Body())
	w.Write(scrapeScript)
	w.Write([]byte("<script type=\"text/javascript\">respond('" + id + "')</script>"))
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
	if validPathRe.FindString(r.URL.Path) == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	//dump, _ := httputil.DumpRequest(r, true)
	log.Println("Handling:", r.Method, r.URL)
	reverseProxy.ServeHTTP(w, r)
}

func response(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get(scrapeToken)
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
		w.Header().Add(scrapeAction, next)
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

	http.HandleFunc("/scrape", scrapeHandler(ch))
	http.HandleFunc("/response", response)
	http.HandleFunc("/", handle)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
