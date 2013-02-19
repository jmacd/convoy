package boards

import "log"
import "io/ioutil"
import "net/http"
import "regexp"
//import "code.google.com/p/go.net/html"

const (
	// TODO(jmacd): Take this from the scraper
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.17 (KHTML, like Gecko) Chrome/24.0.1312.57 Safari/537.17"

	// TODO(jmacd): This is totally specific to the Trulos problem
	// see note below.
	cdataStart = `//<!\[CDATA\[`
	cdataEnd = `//\]\]>`
)

var (
	client *http.Client
	cdataStartRe *regexp.Regexp
	cdataEndRe *regexp.Regexp
)

func init() {
	// Note: Microsoft ASP.NET has a bug in certain browser
	// configurations which causes incorrect receipt of the
	// ScriptResource.axd file.  E.g.,
	// stackoverflow.com/questions/5681122/asp-net-ipad-safari-cache-issue
	// Seems to be a problem _only_ when a proxy is involved.

	client = &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment,
			DisableCompression: true},
	}

	var err1, err2 error
	cdataStartRe, err1 = regexp.Compile(cdataStart)
	cdataEndRe, err2 = regexp.Compile(cdataEnd)

	if err1 != nil || err2 != nil {
		log.Println("Regexp error: ", err1, err2)
	}
}

func GetUrl(host, uri, query string) ([]byte, error) {
	url := "http://" + host + uri + query
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)
	log.Println("Trying", url)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// CDATA sections are converted to comments when setting the innerHTML
// property of a DOM element (at least in Chrome / Firefox).  As a
// workaround, fix it with proper escaping -- but need to assume that
// this only happens inside <script type="text/javascript"> elements.
// stackoverflow.com/questions/7065615/innerhtml-converts-cdata-to-comments
func RepairCDATA(data []byte) (res []byte) {
	for len(data) > 0 {
		start := cdataStartRe.FindIndex(data)
		if start == nil {
			res = append(res, data...)
			return
		}
		res = append(res, data[0:start[0]]...)
		data = data[start[1]:]
		end := cdataEndRe.FindIndex(data)
		if end == nil {
			// Unbalanced case, whatever...
			res = append(res, data...)
			return
		}
		res = append(res, data[0:end[0]]...)
		data = data[end[1]:]
	}
	return
}