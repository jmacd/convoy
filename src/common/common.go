package common

import "crypto/tls"
import "flag"
import "fmt"
import "io/ioutil"
import "log"
import "net/http"
import "os"
import "runtime"
import "runtime/pprof"
import "time"

var mem_profile = flag.Bool("mem_profile", false, "Write memory profiles")
var num_cpu = flag.Int("num_cpu", runtime.NumCPU(), "Number of CPUs")

const (
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) " +
		"AppleWebKit/537.18 (KHTML, like Gecko) Chrome/24.0.1312.58 Safari/537.18"

	sqlDateFmt = "2006-01-02 15:04:05"
	loadDateFmt = "2006-01-02"
)

var (
	client *http.Client
	secure *http.Client
)

func init() {
	// Note: Microsoft ASP.NET has a bug in certain browser
	// configurations which causes incorrect receipt of the
	// ScriptResource.axd file.  E.g.,
	// stackoverflow.com/questions/5681122/asp-net-ipad-safari-cache-issue
	// Seems to be a problem _only_ when a proxy is involved.
	// { DisableCompression: true }

	client = &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment,
			DisableCompression: true,
		},
	}
	secure = &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{RootCAs: nil},
		},
	}
}

func SleepAWhile(url, query string) {
	time.Sleep(time.Second * 2)
}

func ColonPort(p int) string {
	return fmt.Sprint(":", p)
}

func GetUrl(host, uri, query string) ([]byte, error) {
	return getUrlInternal("http", host, uri, query, client, true)
}

func GetUrlFast(host, uri, query string) ([]byte, error) {
	return getUrlInternal("http", host, uri, query, client, false)
}

func GetSecureUrl(host, uri, query string) ([]byte, error) {
	return getUrlInternal("https", host, uri, query, secure, true)
}

func getUrlInternal(
	protocol, host, uri, query string, c *http.Client, addSleep bool) ([]byte, error) {
	if addSleep {
		SleepAWhile(uri, query)
	}
	url := protocol + "://" + host + uri + query
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", UserAgent)
	resp, err := c.Do(req)
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

var profileCount = 0

func PrintMem() {
	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)

	pname := "<none>"
	if *mem_profile {
		pname = fmt.Sprint("prof", profileCount)
		of, err := os.Create(pname)
		profileCount++
		defer of.Close()
		if err == nil {
			pprof.Lookup("heap").WriteTo(of, 1)
		}
	}

	log.Println("Memory allocated:", ms.Alloc,
		"Total:", ms.TotalAlloc, "Sys:", ms.Sys,
		"Goroutines:", runtime.NumGoroutine(),
		"Profile:", pname)
}

func ParseLoadDate(fmt string) (time.Time, error) {
	return time.Parse(loadDateFmt, fmt)
}

func ParseSqlDate(fmt string) (time.Time, error) {
	return time.Parse(sqlDateFmt, fmt)
}

func FormatLoadDate(t time.Time) string {
	return t.Format(loadDateFmt)
}

func NumCPU() int {
	return *num_cpu
}
