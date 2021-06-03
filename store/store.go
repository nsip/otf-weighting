package store

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/digisan/gotk/io"
	"github.com/google/uuid"
)

type (
	SaveOpt struct {
		WG       *sync.WaitGroup
		Dir, Ext string                      // files directory & file extension
		SM       *sync.Map                   // sync map ptr
		M        map[interface{}]interface{} // map
		// more ...
	}
)

func (opt *SaveOpt) file(key, value string) {
	if opt.Dir != "" {
		absdir, _ := io.AbsPath(opt.Dir, false)
		fullpath := filepath.Join(absdir, key) + "." + strings.TrimLeft(opt.Ext, ". \t")
		fullpath = strings.TrimRight(fullpath, ". \t")
		io.MustWriteFile(fullpath, []byte(value))
	}
}

func (opt *SaveOpt) sm(key, value string) {
	if opt.SM != nil {
		opt.SM.Store(key, value)
	}
}

func (opt *SaveOpt) m(key, value string) {
	if opt.M != nil {
		opt.M[key] = value
	}
}

// more saving func ...

func (opt *SaveOpt) batchSave(key, value string) {

	defer func() {
		if opt.WG != nil {
			opt.WG.Done()
		}
	}()
	if opt.WG != nil {
		opt.WG.Add(1)
	}

	opt.file(key, value)
	opt.sm(key, value)
	opt.m(key, value)
	// ... more
}

func (opt *SaveOpt) Wait() {
	if opt.WG != nil {
		opt.WG.Wait()
	}
}

///////////////////////////////////////////////////////

func (opt *SaveOpt) Save(key, value string) {
	opt.batchSave(key, value)
}

func (opt *SaveOpt) IDXSaveFactory(start int) func(value string) {
	idx := int64(start)
	return func(value string) {
		opt.batchSave(fmt.Sprintf("%04d", atomic.AddInt64(&idx, 1)), value)
	}
}

func (opt *SaveOpt) TSSave(value string) {

	opt.batchSave(time.Now().Format("2006-01-02 15:04:05.000000"), value)

	// current := time.Now()
	// // StampMicro
	// fmt.Println("yyyy-mm-dd HH:mm:ss: ", current.Format("2006-01-02 15:04:05.000000"))
	// // yyyy-mm-dd HH:mm:ss:  2016-09-02 15:53:07.159994
	// // StampNano
	// fmt.Println("yyyy-mm-dd HH:mm:ss: ", current.Format("2006-01-02 15:04:05.000000000"))
	// // yyyy-mm-dd HH:mm:ss:  2016-09-02 15:53:07.159994437
}

func (opt *SaveOpt) GUIDSave(value string) {
	opt.batchSave(strings.ReplaceAll(uuid.New().String(), "-", ""), value)
}
