package main

import "database/sql"
import "io/ioutil"
import "log"
import "net/http"
import "net/http/httputil"
import "net/url"
import "regexp"
import "sync"
import "time"
import _ "github.com/Go-SQL-Driver/MySQL"

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

// type cachedContent struct {
// }

type proxyTransport struct {
	transport *http.Transport
//	scriptCache map[string]*cachedContent
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
		proxyRequest,
		&proxyTransport{&http.Transport{
				Proxy:              proxyUrl,
				DisableCompression: true},
		},
		time.Duration(0)}

	validPathRe = regexp.MustCompile(validPathRegexp)
}

func (p *proxyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := p.transport.RoundTrip(r)
	//dump, _ := httputil.DumpResponse(resp, true)
	//log.Println("Handling response:", r.Method, r.URL, len(dump), "bytes")
	return resp, err
}

func proxyUrl(r *http.Request) (*url.URL, error) {
	// Remove Referer b/c is added after proxyRequest():
	r.Header.Del("X-Forwarded-For")
	return http.ProxyFromEnvironment(r)
}

func proxyRequest(r *http.Request) {
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
	//log.Println("Handing work to a scraper", page)
	w.Header().Add(scrapeToken, id)
	w.Write(page.Body())
	w.Write(scrapeScript)

	// The initial response callback.
	w.Write([]byte("<script type=\"text/javascript\">respond('" + 
		id + "')</script>"))

	// The __doPostBack response callback.
	// http://stackoverflow.com/questions/6504472/how-to-wait-on-the-dopostback-method-to-complete-in-javascript
	w.Write([]byte("<script type=\"text/javascript\">" +
		"Sys.WebForms.PageRequestManager.getInstance()." +
		"add_endRequest(function() { respond('" + id + "') })" + 
		"</script>"))
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
	//log.Println("Handling request:", r.Method, r.URL, len(dump), "bytes")
	boards.SleepAWhile(r.URL.Path, r.URL.RawQuery)
	reverseProxy.ServeHTTP(w, r)
}

func response(w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get(scrapeToken)
	//log.Println("Scraper finished work", id)
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
		//log.Println("Sending next action", next)
		w.Header().Add(scrapeAction, next)
	} else {
		smutex.Lock()
		delete(smap, id)
		smutex.Unlock()
	}
}

// loadBoard produces items for scraping on the channel.
func loadBoard(pageCh chan<- scraper.Page, loadCh chan<- *boards.Load) error {
	tt, err := boards.NewTrulos(loadCh)
	if err != nil {
		return err
	}
	if err := tt.Init(); err != nil {
		return err
	}
	go tt.Read(pageCh)
	return nil
}

// openDb opens and tests the database connection.
func openDb() (*sql.DB, error) {
	conn, err := sql.Open("mysql", 
		"test:@/Convoy?charset=utf8")
	if err != nil {
		return conn, err
	}
	// Test that the connection is good; because the driver call
	// to open the database is defered until the first request.
	_, err = conn.Exec("SELECT 1;")
	if err != nil {
		log.Fatal("Database not opened!", err)
	}
	return conn, err
}

func saveLoad(stmt *sql.Stmt, load *boards.Load) error {
	_, err := stmt.Exec(
		load.PickupDate,
		load.OriginState,
		load.OriginCity,
		load.DestState,
		load.DestCity,
		load.LoadType,
		load.Length,
		load.Weight,
		load.Equipment,
		load.Price,
		load.Stops,
		load.Phone)
	return err
}

func processLoads(conn *sql.DB, loadCh <-chan *boards.Load) {
	stmt, err := conn.Prepare(
		"INSERT INTO Convoy.TruckLoads " +
		"(PickupDate, OriginState, OriginCity, " +
		"DestState, DestCity, LoadType, Length, " +
		"Weight, Equipment, Price, Stops, Phone) " +
		" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal("Could not prepare INSERT statement")
	}
	for load := range loadCh {
		if err := saveLoad(stmt, load); err != nil {
			log.Print("Could not save load: ", err)
		}
	}
}

func main() {
	conn, err := openDb()
	if err != nil {
		log.Fatal("Couldn't connect to database: ", err)
	}
	defer conn.Close()
	ch1 := make(chan scraper.Page)
	ch2 := make(chan *boards.Load)
	if err := loadBoard(ch1, ch2); err != nil {
		log.Fatal("Couldn't initialize load board: ", err)
	}
	go processLoads(conn, ch2)

	http.HandleFunc("/scrape", scrapeHandler(ch1))
	http.HandleFunc("/response", response)
	http.HandleFunc("/", handle)

	log.Fatal(http.ListenAndServe(":8000", nil))
}
