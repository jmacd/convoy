package scraper

import "flag"
import "fmt"
import "io"
import "log"
import "net/http"
import "strings"
import "os/exec"
import "path"
import "time"

var xvfbPath = flag.String("xvfb_path", "/usr/bin/Xvfb", 
	"Path for Xvfb")
var browserPath = flag.String("browser_path", "/usr/bin/google-chrome", 
	"Path for a browser")
var debugBrowser = flag.Bool("debug_browser", false, "")
var debugSubprocs = flag.Bool("debug_subprocs", false, "")

type Browser struct {
	server        *http.Server
	xvfb, browse  *exec.Cmd
}

type pipeData struct {
	cmd, data string
	err error
}

func colonPort(p int) string {
	return fmt.Sprint(":", p)
}

func readOut(cmd string, f io.ReadCloser, ch chan<- pipeData) {
	cmd = path.Base(cmd)
	for {
		data := make([]byte, 512)
		b, err := f.Read(data)
		if err == io.EOF {
			time.Sleep(time.Second)
			continue
		}
		if b > 0 {
			ch <- pipeData{cmd, string(data[0:b]), nil}
		}
		if err != nil {
			ch <- pipeData{cmd, "", err}
		}
	}
}

func printOut(ch <-chan pipeData) {
	for {
		select {
		case pd := <- ch:
			if pd.err != nil {
				log.Print(pd.cmd, ":ERROR: ", pd.err)
				break
			} 
			for _, x := range strings.Split(pd.data, "\n") {
				if len(x) > 0 {
					log.Print(pd.cmd, ": ", x)
				}
			}
		}
	}
}

func startProcess(cmd string, env []string, args ...string) (*exec.Cmd, error) {
	proc := exec.Command(cmd, args...)
	proc.Env = env
	var err error
	if *debugSubprocs {
		var perr, pout io.ReadCloser
		if perr, err = proc.StderrPipe(); err != nil {
			return nil, err
		}
		if pout, err = proc.StdoutPipe(); err != nil {
			return nil, err
		}
		ch := make(chan pipeData)
		go readOut(cmd, pout, ch)
		go readOut(cmd, perr, ch)
		go printOut(ch)
	}
	if err = proc.Start(); err != nil {
		return nil, err
	}
	return proc, nil
}

func NewBrowser(httpPort, xPortOffset int, 
	startUri string, handler http.Handler) (*Browser, error) {
	b := &Browser{}
	server := &http.Server{
		colonPort(httpPort),
		handler,
		time.Hour,
		time.Hour,
		1 << 20,
		nil, nil}
	b.server = server
	go func() {
		log.Fatal(server.ListenAndServe())
	}()
	display := colonPort(xPortOffset)
	xvfb, err := startProcess(*xvfbPath, []string{"NOENV=yes"}, display)
	if err != nil {
		return nil, err
	}
	b.xvfb = xvfb
	if !*debugBrowser {
		browse, err := startProcess(*browserPath, 
			[]string{"DISPLAY=" + display},
			fmt.Sprint("http://localhost:", httpPort, startUri))
		if err != nil {
			return nil, err
		}
		b.browse = browse
	}
	return b, nil
}

func (b *Browser) Cleanup() {
	if b.xvfb != nil {
		b.xvfb.Process.Kill()
	}
	if b.browse != nil {
		b.browse.Process.Kill()
	}
}
