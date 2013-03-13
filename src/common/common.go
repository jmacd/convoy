package common

import "net/http"
import "io/ioutil"
import "time"

const (
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) " +
		"AppleWebKit/537.18 (KHTML, like Gecko) Chrome/24.0.1312.58 Safari/537.18"
)

var (
	client      *http.Client
)

func init() {
	// Note: Microsoft ASP.NET has a bug in certain browser
	// configurations which causes incorrect receipt of the
	// ScriptResource.axd file.  E.g.,
	// stackoverflow.com/questions/5681122/asp-net-ipad-safari-cache-issue
	// Seems to be a problem _only_ when a proxy is involved.

	client = &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment,
			/* DisableCompression: true */},
	}
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
