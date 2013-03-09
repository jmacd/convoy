package boards

import "log"
import "io/ioutil"
import "net/http"
import "regexp"
import "time"
import "strconv"

const (
	// TODO(jmacd): Take this from the scraper
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) " +
		"AppleWebKit/537.17 (KHTML, like Gecko) Chrome/24.0.1312.57 Safari/537.17"
)

var (
	client      *http.Client
	httpColonRe *regexp.Regexp
	fpnumRe     *regexp.Regexp
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

	httpColonRe = regexp.MustCompile("https?:/")
	fpnumRe = regexp.MustCompile(`^$?([0-9]+(?:\.[0-9]+)?)`)
}

func SleepAWhile(url, query string) {
	time.Sleep(time.Second * 10)
}

func GetUrl(host, uri, query string) ([]byte, error) {
	SleepAWhile(uri, query)
	url := "http://" + host + uri + query
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", UserAgent)
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

func HijackExternalRefs(data []byte) []byte {
	return httpColonRe.ReplaceAll(data, []byte{})
}

func ParseLeadingInt(s string) int {
	if len(s) == 0 {
		return 0
	}
	m := fpnumRe.FindString(s)
	if len(m) != 0 {
		f, _ := strconv.ParseFloat(m, 64)
		return int(f)
	}
	log.Print("Could not parse number: ", s)
	return 0
}
