// Module for reading load information from trulos.com

package boards

import "bytes"
import "flag"
import "fmt"
import "log"
import "net/url"
import "regexp"
import "strconv"
import "strings"
import "time"
import "code.google.com/p/go.net/html"
import "code.google.com/p/go.net/html/atom"

import "common"
import "scraper"

var stateRe = flag.String("trulos_state_regexp", ".*", "")

const (
	baseUri   = "/Trulos/Post-Truck-Loads/Truck-Load-Board.aspx"
	contentId = "ContentPlaceHolder1_GridView1"
	// Regexp for finding actions in HTML-escaped javascript
	escapedRegexp = `[a-zA-Z0-9$&#;,]+`
	pageRegexp    = `__doPostBack\(` + escapedRegexp +
		`Page` + escapedRegexp + `\)`
)

type trulosBoard struct {
	host        string
	stateUriRe  *regexp.Regexp
	equipRe     *regexp.Regexp
	pageRe      *regexp.Regexp
	stateProcRe *regexp.Regexp
	loadf       func([]*Load) error
	states      []*trulosState
}

type trulosState struct {
	board          *trulosBoard
	name           string
	uri            string
	equipmentTypes []string
}

type trulosScrape struct {
	state   *trulosState
	equip   string
	body    []byte
	actions []string
	compCh  chan<- int
	respCh  chan *scraper.Result
	loads   []*Load
}

func NewTrulos(loadf func([]*Load) error) (LoadBoard, error) {
	stateUriRe := regexp.MustCompile(regexp.QuoteMeta(baseUri+"?STATE=") + `(\w+)`)
	equipRe := regexp.MustCompile(`\?STATE=(?:\w+)&amp;Equipment=([ /\w]+)`)
	pageRe := regexp.MustCompile(pageRegexp)
	stateProcRe := regexp.MustCompile(*stateRe)
	board := &trulosBoard{"www.trulos.com",
		stateUriRe, equipRe, pageRe, stateProcRe,
		loadf, nil}
	return board, nil
}

func (t *trulosBoard) Init() error {
	body, err := common.GetUrl(t.host, "", "")
	if err != nil {
		return err
	}
	links := t.stateUriRe.FindAllStringSubmatch(string(body), -1)
	for _, si := range links {
		t.states = append(t.states, &trulosState{t, si[1], si[0], nil})
	}
	return nil
}

func (s *trulosState) getEquipmentTypes() {
	body, err := common.GetUrl(s.board.host, s.uri, "")
	if err != nil {
		log.Print("No equipment types found", s)
		return
	}
	links := s.board.equipRe.FindAllStringSubmatch(string(body), -1)
	for _, si := range links {
		s.equipmentTypes = append(s.equipmentTypes, si[1])
	}
	//log.Print("equipment types", s.equipmentTypes)
}

func (s *trulosState) queryForEquip(equip string) string {
	return "?STATE=" + s.name + "&Equipment=" + url.QueryEscape(equip)
}

// Read asynchronously reads pages from the board and passes them to the
// scrape-evaluator.
func (t *trulosBoard) Read(pages chan<- scraper.Page) {
	compCh := make(chan int)
	for _, state := range t.states {
		if len(t.stateProcRe.FindString(state.name)) == 0 {
			continue
		}
		state.getEquipmentTypes()
		for _, equip := range state.equipmentTypes {
			//log.Println("Reading Trulos state",
			//            state.name, equip)
			query := state.queryForEquip(equip)
			body, err := common.GetUrl(t.host, baseUri, query)
			if err != nil {
				log.Print("Problem reading Trulos", query)
				continue
			}
			actions := state.board.pageRe.FindAllString(
				string(body), -1)
			for i := 0; i < len(actions); i++ {
				actions[i] = html.UnescapeString(actions[i])
			}
			scrape := &trulosScrape{state, equip,
				HijackExternalRefs(body), actions,
				compCh, make(chan *scraper.Result),
				nil}
			// Process len(actions) + 1 pages, block until
			// completion.
			go scrape.ProcessScrapes()
			pages <- scrape
			<-compCh
		}
	}
}

func (s *trulosScrape) ProcessScrapes() {
	for i := 0; i <= len(s.actions); i++ {
		s.Process(<-s.respCh)
	}
	if err := s.state.board.loadf(s.loads); err != nil {
		log.Printf("Error writing %d loads: %s: %s",
			len(s.loads), s, err)
	} else {
		log.Printf("Wrote %d loads: %s", len(s.loads), s)
	}
	s.compCh <- 1
}

func (s *trulosScrape) Id() string {
	return fmt.Sprint("Trulos-", s.state.name, "-", s.equip)
}

func (s *trulosScrape) Body() []byte {
	return s.body
}

func (s *trulosScrape) Actions() []string {
	return s.actions
}

func (s *trulosScrape) Channel() chan<- *scraper.Result {
	return s.respCh
}

func (s *trulosScrape) Process(r *scraper.Result) {
	//log.Print("Scrape result for [", string(r.Action), "]: ", s, " ",
	//	len(r.Data), " bytes")
	doc, err := html.Parse(bytes.NewReader(r.Data))
	if err != nil {
		log.Print("Scrape parse error", s.state.name, s.equip, err)
		return
	}
	s.TraverseHTML(doc)
}

func attrIdIs(n *html.Node, value string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "id" && attr.Val == value {
			return true
		}
	}
	return false
}

func (s *trulosScrape) TraverseContentTable(n *html.Node, depth int) {
	// Level 0 is TABLE
	// Level 1 is TBODY
	// Level 2 is TR
	if depth == 2 && n.Type == html.ElementNode && n.DataAtom == atom.Tr {
		row := s.TraverseContentRow(n, depth, nil)
		s.ProcessRowData(row)
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			s.TraverseContentTable(c, depth+1)
		}
	}
}

func (s *trulosScrape) TraverseContentRow(n *html.Node, depth int,
	data []string) []string {
	// Level 3 is TD
	// Level 4 is FONT
	// Level 5 and higher are target data
	if depth == 3 && n.DataAtom == atom.Th {
		return data
	}
	if depth > 3 && n.Type == html.TextNode {
		data = append(data, n.Data)
	} else {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			data = s.TraverseContentRow(c, depth+1, data)
		}
	}
	return data
}

func (s *trulosScrape) TraverseHTML(n *html.Node) {
	if n.Type == html.ElementNode && n.DataAtom == atom.Table &&
		attrIdIs(n, contentId) {
		s.TraverseContentTable(n, 0)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.TraverseHTML(c)
	}
}

func (s *trulosScrape) ProcessRowData(row []string) {
	var trimmed []string
	for _, item := range row {
		trimmed = append(trimmed, strings.TrimSpace(item))
	}
	// We index up to trimmed[15]
	if len(row) < 16 {
		return
	}
	dateStr := trimmed[0]
	dateSplit := strings.Split(dateStr, "/")
	if len(dateSplit) != 3 {
		log.Printf("Bad date: %s for %s: %q",
			dateStr, s, trimmed)
		return
	}
	dateYear, _ := strconv.Atoi(dateSplit[2])
	dateMonth, _ := strconv.Atoi(dateSplit[0])
	dateDay, _ := strconv.Atoi(dateSplit[1])
	date := time.Date(dateYear, time.Month(dateMonth),
		dateDay, 12, 0, 0, 0, time.Local)
	origin := trimmed[2]
	if strings.ToUpper(trimmed[4]) != s.state.name {
		log.Printf("Unexpected state: %s for %s: %q",
			trimmed[4], s, trimmed)
		return
	}
	destCity, destState := trimmed[6], strings.ToUpper(trimmed[7])
	loadType := trimmed[8]
	llen := ParseLeadingInt(trimmed[9])
	if trimmed[10] != s.equip {
		log.Printf("Unexpected equipment type: %s for %s: %q",
			trimmed[10], s, trimmed)
		return
	}
	price := ParseLeadingInt(trimmed[11])
	weight := ParseLeadingInt(trimmed[12])
	stops := ParseLeadingInt(trimmed[13])
	if price < 10 {
		price = 0 // TODO(jmacd): Assume per mile, fix.
	}
	if weight < 100 {
		weight *= 1000 // Assume per thousand pounds
	}
	phone := trimmed[15]
	load := &Load{date, common.ProperName(origin), s.state.name,
		common.ProperName(destCity), destState,
		loadType, llen, weight, s.equip, price, stops, phone}
	s.loads = append(s.loads, load)
}

func (t *trulosBoard) String() string {
	return fmt.Sprintf("%s -> %s", t.host, t.states)
}

func (s *trulosState) String() string {
	return fmt.Sprintf("%s [%s]", s.name, s.uri)
}

func (s *trulosScrape) String() string {
	return s.Id()
}
