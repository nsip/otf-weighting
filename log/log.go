package log

import (
	"fmt"
	"sync/atomic"

	lk "github.com/digisan/logkit"
)

func Factory4IdxLog(start int) func(v ...interface{}) {
	index := int64(start - 1)
	return func(v ...interface{}) {
		prefix := fmt.Sprintf("%05d - ", atomic.AddInt64(&index, 1))
		content := fmt.Sprint(v...)
		lk.Log("%s%s", prefix, content)
	}
}
