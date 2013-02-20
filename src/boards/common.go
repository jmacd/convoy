package boards

import "log"
import "io/ioutil"
import "net/http"
import "regexp"

const (
	// TODO(jmacd): Take this from the scraper
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) " +
		"AppleWebKit/537.17 (KHTML, like Gecko) Chrome/24.0.1312.57 Safari/537.17"
)

var (
	client      *http.Client
	httpColonRe *regexp.Regexp
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

func HijackExternalRefs(data []byte) []byte {
	return httpColonRe.ReplaceAll(data, []byte{})
}
