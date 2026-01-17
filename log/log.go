package log

import (
	"fmt"
	"time"
	"runtime"
	"path/filepath"
)

func Println(args ...any) {
	log(getCallerSource(runtime.Caller(1)), fmt.Sprint(args...))
}

func Printf(format string, args ...any) {
	log(getCallerSource(runtime.Caller(1)), format, args...)
}

func getCallerSource(pc uintptr, file string, line int, ok bool) string {
	return filepath.Base(filepath.Dir(file))
}

func log(source, message string, args ...any) {
	h, m, s := time.Now().Clock()
	fmt.Printf("%02v:%02v:%02v %v: %v\n", h, m, s, source, fmt.Sprintf(message, args...))
}
