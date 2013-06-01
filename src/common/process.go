package common
import "io"
import "time"
import "flag"
import "os/exec"
import "path"
import "log"
import "strings"

var debugSubprocs = flag.Bool("debug_subprocs", false, "")

type Cmd interface {
	Cleanup()
}

type impl struct {
	*exec.Cmd
}

type pipeData struct {
	cmd, data string
	err       error
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
		case pd := <-ch:
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

func (c impl) Cleanup() {
	if c.Cmd != nil {
		c.Cmd.Process.Kill()	
	}
}

func StartProcess(cmd string, env []string, args ...string) (Cmd, error) {
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
	return impl{proc}, nil
}
