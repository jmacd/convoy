package main

import "bytes"
import "database/sql"
import "flag"
import "fmt"
import "io/ioutil"
import "log"
import "net/http"
import "net/http/httputil"
import "net/url"
import "regexp"
import "strings"
import "sync"
import "time"

import "boards"
import "common"
import "data"
import "scraper"

const (
	scrapeToken     = "Scraper-Token"
	scrapeAction    = "Scraper-Action"
	scrapeJsFile    = "scraper/scraper.js"
	validPathRegexp = `^/(Trulos|Truck).*`
)

type scrapeState struct {
	page scraper.Page
	aid  int // Index of the current page.Action() or -1
}

type cachedContent struct {
	resp http.Response
	body []byte
}

type proxyTransport struct {
	transport   *http.Transport
	scriptCache map[string]*cachedContent
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
			make(map[string]*cachedContent),
		},
		time.Duration(0)}

	validPathRe = regexp.MustCompile(validPathRegexp)
}

func (p *proxyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var cacheKey string
	if strings.HasSuffix(r.URL.Path, ".axd") {
		cacheKey = fmt.Sprint(r.Method, r.URL)
		if cached, has := p.scriptCache[cacheKey]; has {
			resp := new(http.Response)
			*resp = cached.resp
			resp.Body = ioutil.NopCloser(
				bytes.NewReader(cached.body))
			//log.Println("Cache hit!", cacheKey)
			return resp, nil
		}
	}

	resp, err := p.transport.RoundTrip(r)
	if err != nil {
		return resp, err
	}

	if len(cacheKey) != 0 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return resp, err
		}
		cc := new(cachedContent)
		cc.body = body
		cc.resp = *resp
		cc.resp.Body = nil
		resp.Body = ioutil.NopCloser(bytes.NewReader(cc.body))
		p.scriptCache[cacheKey] = cc
		//log.Print("Did not sleep for cacheable content: ",
		//	r.URL.Path, r.URL.RawQuery)
	} else {
		common.SleepAWhile(r.URL.Path, r.URL.RawQuery)
	}

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
	r.Header.Del("User-Agent")
	r.Header.Set("User-Agent", common.UserAgent)
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
	page.Channel() <- &scraper.Result{act, body, err}

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
func loadBoard(loadf func([]*boards.Load) error) (boards.LoadBoard, error) {
	tt, err := boards.NewTrulos(loadf)
	if err != nil {
		return nil, err
	}
	if err := tt.Init(); err != nil {
		return nil, err
	}
	return tt, nil
}

func saveLoad(stmt *sql.Stmt, scrapeId int64, load *boards.Load) error {
	_, err := stmt.Exec(
		scrapeId,
		load.PickupDate,
		load.Origin.State,
		load.Origin.City,
		load.Dest.State,
		load.Dest.City,
		load.LoadType,
		load.Length,
		load.Weight,
		load.Equipment,
		load.Price,
		load.Stops,
		load.Phone)
	return err
}

func processLoads(stmt *sql.Stmt, scrapeId int64,
	loads []*boards.Load) error {
	for _, load := range loads {
		if err := saveLoad(stmt, scrapeId, load); err != nil {
			return err
		}
	}
	return nil
}

func startScrape(pageCh chan<- scraper.Page, quitCh chan<- int) {
	conn, err := data.OpenDb()
	if err != nil {
		log.Fatal("Couldn't connect to database: ", err)
	}
	result, err := conn.Exec("INSERT INTO " + data.Table("Scrapes") +
		" (StartTime) VALUES (NOW())")
	if err != nil {
		log.Fatal("Could not insert new Scrape: ", err)
	}
	scrapeId, err := result.LastInsertId()
	if err != nil {
		log.Fatal("Insert did not yield a ScrapeId: ", err)
	}
	stmt, err := conn.Prepare(
		"INSERT INTO " + data.Table("TruckLoads") +
			" (ScrapeId, PickupDate, OriginState, OriginCity, " +
			"DestState, DestCity, LoadType, Length, " +
			"Weight, Equipment, Price, Stops, Phone) " +
			" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal("Could not prepare INSERT statement")
	}
	board, err := loadBoard(func(loads []*boards.Load) error {
		return processLoads(stmt, scrapeId, loads)
	})
	if err != nil {
		log.Fatal("Couldn't initialize load board: ", err)
	}
	log.Print("Starting ", data.Table("ScrapeId"), " = ", scrapeId)
	board.Read(pageCh)
	_, err = conn.Exec("UPDATE "+data.Table("Scrapes")+
		" SET FinishTime = NOW() "+
		"WHERE ScrapeId = ?", scrapeId)
	conn.Close()
	quitCh <- 1
}

func startServer(pageCh <-chan scraper.Page) (*scraper.Browser, error) {
	http.HandleFunc("/scrape", scrapeHandler(pageCh))
	http.HandleFunc("/response", response)
	http.HandleFunc("/", handle)

	return scraper.NewBrowser("/scrape", http.DefaultServeMux)
}

func main() {
	flag.Parse()
	pageCh := make(chan scraper.Page)
	quitCh := make(chan int)

	browser, err := startServer(pageCh)
	if err != nil {
		log.Fatalln("Failed to start HTTP server", err)
	}
	defer browser.Cleanup()

	go startScrape(pageCh, quitCh)

	<-quitCh
}
