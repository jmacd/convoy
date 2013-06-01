package scraper

import "flag"
import "fmt"
import "log"
import "net/http"
import "time"

import "common"

var browserPath = flag.String("browser_path", "/usr/bin/google-chrome", "Path for a browser")
var debugBrowser = flag.Bool("debug_browser", false, "")
var httpPort = flag.Int("http_port", 8000, "")
var xvfbPath = flag.String("xvfb_path", "/usr/bin/Xvfb", "Path for Xvfb")
var xvfbPortOffset = flag.Int("xvfb_port_offset", 1, "")

type Browser struct {
	server       *http.Server
	xvfb, browse common.Cmd
}

func NewBrowser(startUri string, handler http.Handler) (*Browser, error) {
	b := &Browser{}
	server := &http.Server{
		common.ColonPort(*httpPort),
		handler,
		time.Hour,
		time.Hour,
		1 << 20,
		nil, nil}
	b.server = server
	go func() {
		log.Fatal(server.ListenAndServe())
	}()
	display := common.ColonPort(*xvfbPortOffset)
	xvfb, err := common.StartProcess(*xvfbPath, []string{"NOENV=yes"}, display)
	if err != nil {
		return nil, err
	}
	b.xvfb = xvfb
	if !*debugBrowser {
		browse, err := common.StartProcess(*browserPath,
			[]string{"DISPLAY=" + display},
			fmt.Sprint("http://localhost:", *httpPort, startUri))
		if err != nil {
			return nil, err
		}
		b.browse = browse
	}
	return b, nil
}

func (b *Browser) Cleanup() {
	if b.xvfb != nil {
		b.xvfb.Cleanup()
	}
	if b.browse != nil {
		b.browse.Cleanup()
	}
}
