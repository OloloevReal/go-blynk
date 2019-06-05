package simlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
)

/*
Just simple logs, not more
*/

var version = "0.0.2"

var levels = []string{"DEBUG", "INFO", "ERROR", "PANIC", "FATAL"}

type Logger struct {
	out    io.Writer
	debug  bool
	caller bool
}

type Option func(l *Logger)

func SetDebug(l *Logger) {
	l.debug = true
}

func SetCaller(l *Logger) {
	l.caller = true
}

func NewLogger(opts ...Option) *Logger {
	l := &Logger{out: os.Stderr, debug: false, caller: false}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// func (l *Logger) Printf(format string, v ...interface{}) {
// 	l.logf(log.Printf, format, v...)
// }

type Func func(format string, v ...interface{})

func (l *Logger) SetOutput(w io.Writer) {
	log.SetOutput(w)
}

func (l *Logger) logf(fn Func, format string, v ...interface{}) {
	lv, msg := l.extractLevel(fmt.Sprintf(format, v...))
	if lv == "DEBUG" && !l.debug {
		return
	}

	//runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if !l.caller {
		fn("[%s] %s", lv, msg)
	} else {
		ci := l.reportCaller(0)
		fn("[%s] {%s} %s", lv, fmt.Sprintf("%s:%d %s", ci.File, ci.Line, ci.FuncName), msg)
	}

}

func (l *Logger) logln(fn func(v ...interface{}), v ...interface{}) {
	lv, msg := l.extractLevel(fmt.Sprint(v...))
	if lv == "DEBUG" && !l.debug {
		return
	}

	if !l.caller {
		fn(fmt.Sprintf("[%s] %s", lv, msg))
	} else {
		ci := l.reportCaller(0)
		fn(fmt.Sprintf("[%s] {%s} %s", lv, fmt.Sprintf("%s:%d %s", ci.File, ci.Line, ci.FuncName), msg))
	}
}

func (l *Logger) extractLevel(line string) (level, msg string) {
	for _, lv := range levels {
		if strings.HasPrefix(line, "["+lv+"]") {
			return lv, line[len(lv)+3:]
		}
	}
	return "INFO", line
}

var globalLog = NewLogger()

func Printf(format string, v ...interface{}) {
	globalLog.logf(log.Printf, format, v...)
}

func Println(v ...interface{}) {
	globalLog.logln(log.Println, v...)
}

func Fatalf(format string, v ...interface{}) {
	globalLog.logf(log.Fatalf, format, v...)
}

func Fatalln(v ...interface{}) {
	globalLog.logln(log.Fatalln, v...)
}

func Panicf(format string, v ...interface{}) {
	globalLog.logf(log.Panicf, format, v...)
}

func Panicln(v ...interface{}) {
	globalLog.logln(log.Panicln, v...)
}

func SetOptions(opts ...Option) {
	for _, opt := range opts {
		opt(globalLog)
	}
}

func GetDafault() *Logger {
	return globalLog
}

type callerInfo struct {
	File     string
	Line     int
	FuncName string
	Pkg      string
}

// calldepth 0 identifying the caller of reportCaller()
func (l *Logger) reportCaller(calldepth int) (res callerInfo) {
	//{svc/handler.go:101 h.MyFunc1}
	// caller gets file, line number abd function name via runtime.Callers
	// file looks like /go/src/github.com/go-pkgz/lgr/logger.go
	// file is an empty string if not known.
	// funcName looks like:
	//   main.Test
	//   foo/bar.Test
	//   foo/bar.Test.func1
	//   foo/bar.(*Bar).Test
	//   foo/bar.glob..func1
	// funcName is an empty string if not known.
	// line is a zero if not known.
	caller := func(calldepth int) (file string, line int, funcName string) {
		pcs := make([]uintptr, 1)
		n := runtime.Callers(calldepth, pcs)
		if n != 1 {
			return "", 0, ""
		}

		frame, _ := runtime.CallersFrames(pcs).Next()

		return frame.File, frame.Line, frame.Function
	}

	// add 5 to adjust stack level because it was called from 3 nested functions added by lgr, i.e. caller,
	// reportCaller and logf, plus 2 frames by runtime
	filePath, line, funcName := caller(calldepth + 2 + 3)
	if (filePath == "") || (line <= 0) || (funcName == "") {
		return callerInfo{}
	}

	_, pkgInfo := path.Split(path.Dir(filePath))
	res.Pkg = pkgInfo

	res.File = filePath
	if pathElems := strings.Split(filePath, "/"); len(pathElems) > 2 {
		res.File = strings.Join(pathElems[len(pathElems)-2:], "/")
	}
	res.Line = line

	funcNameElems := strings.Split(funcName, "/")
	res.FuncName = funcNameElems[len(funcNameElems)-1]

	return res
}
