package log

import (
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
)

var (
	fSf = fmt.Sprintf
)

// type StackTrace struct {
// 	msg  string
// 	path string
// }

// // New function constructs a new `StackTrace` struct by using given panic
// // message, absolute path of the caller file and the line number.
// func New(msg string) *StackTrace {
// 	_, file, line, _ := runtime.Caller(1)
// 	p, _ := os.Getwd()

// 	return &StackTrace{
// 		msg:  msg,
// 		path: fmt.Sprintf("%s:%d", strings.TrimPrefix(file, p), line),
// 	}
// }

// func (s *StackTrace) Message() string {
// 	return s.msg
// }

// func (s *StackTrace) Path() string {
// 	return s.path
// }

// lvl: 0. where `running trackCaller(...) in its caller, such as TrackCaller, Caller`
func trackCaller(lvl int) (string, int, string) {
	lvl += 2
	pc := make([]uintptr, 15)
	n := runtime.Callers(lvl, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	return frame.File, frame.Line, frame.Function
}

func Factory4IdxLog(start int) func(v ...interface{}) {
	log.SetFlags(0)
	index := int64(start - 1)
	return func(v ...interface{}) {
		f, l, fn := trackCaller(1)
		prefix1 := fSf("- %05d -", atomic.AddInt64(&index, 1))
		prefix2 := fSf("%s:%d:%s -", f, l, fn)
		v = append([]interface{}{prefix1, prefix2}, v...)
		log.Println(v...)
	}
}
