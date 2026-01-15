package log

import (
	"io"
	"os"
	"fmt"
	"time"
	"sync"
)

var Sys Channel
var Crash Channel

type Channel struct {
	formatter func(string)string
	sinks []io.Writer
}

func (c Channel) WriteString(str string) (int, error) {
	data := []byte(c.formatter(str) + "\n")
	for _, sink := range c.sinks {
		sink.Write(data)
	}
	return len(str), nil
}

func (c Channel) Print(args ...any) {
	c.WriteString(fmt.Sprint(args...))
}

func (c Channel) Printf(format string, args ...any) {
	c.WriteString(fmt.Sprintf(format, args...))
}

func init() {
	t := time.Now()
	y, mn, d := t.Date()
	h, m, s := t.Clock()

	rootDir := fmt.Sprintf("logs/%04v-%02v-%02v_%02v.%02v.%02v", y, int(mn), d, h, m, s)
	os.MkdirAll(rootDir, 0666)

	Sys.formatter = timeFormatter
	Sys.sinks = append(Sys.sinks, os.Stdout)
	Sys.sinks = append(Sys.sinks, newLazyFileWriter(rootDir + "/sys.txt"))

	Crash.formatter = nopFormatter
	Crash.sinks = append(Crash.sinks, os.Stderr)
	Crash.sinks = append(Crash.sinks, newLazyFileWriter(rootDir + "/crash.txt"))
}

func nopFormatter(s string) string {
	return s
}

func timeFormatter(str string) string {
	h, m, s := time.Now().Clock()
	return fmt.Sprintf("%02v.%02v.%02v %v", h, m, s, str)
}

type lazyFile struct {
	value *os.File
	mu sync.Mutex
}

type lazyFileWriter struct {
	path string
	file *lazyFile
}

func newLazyFileWriter(path string) lazyFileWriter {
	return lazyFileWriter{ path: path, file: &lazyFile{} }
}

func (lazy lazyFileWriter) Write(b []byte) (int, error) {
	lazy.file.mu.Lock()
	defer lazy.file.mu.Unlock()

	if lazy.file.value == nil {
		var e error
		lazy.file.value, e = os.OpenFile(lazy.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
		if e != nil {
			return 0, e
		}
	}

	return lazy.file.value.Write(b)
}
