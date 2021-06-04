package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/digisan/gotk/io"
	"github.com/google/uuid"
)

type (
	SaveOpt struct {
		Dir, Ext       string                                       // file directory & file extension
		OnFileConflict func(existing, coming string) (bool, string) // file conflict solver
		SM             *sync.Map                                    // sync map ptr
		OnSMapConflict func(existing, coming string) (bool, string) // sync map conflict solver
		M              map[interface{}]interface{}                  // map
		OnMapConflict  func(existing, coming string) (bool, string) // map conflict solver

		// more ...
		WG  *sync.WaitGroup
		Mtx *sync.Mutex
	}
)

func (opt *SaveOpt) file(key, value string) {
	if opt.Dir != "" {
		absdir, _ := io.AbsPath(opt.Dir, false)
		fullpath := filepath.Join(absdir, key) // full abs file name path without extension
		ext := strings.TrimLeft(opt.Ext, ".")  // extension without prefix '.'
		prevpath := ""

		// record duplicate key number in fullpath as .../key(number).ext
		if matches, err := filepath.Glob(fullpath + "(*)." + ext); err == nil {
			if len(matches) > 0 {
				prevpath = matches[0]
				fname := filepath.Base(prevpath)
				pO, pC := strings.Index(fname, "("), strings.Index(fname, ")")
				num, _ := strconv.Atoi(fname[pO+1 : pC])
				fullpath = filepath.Join(absdir, fmt.Sprintf("%s(%d)", fname[:pO], num+1))
			} else {
				fullpath = fmt.Sprintf("%s(1)", fullpath)
			}
		}

		// add extension
		fullpath = strings.TrimRight(fullpath+"."+ext, ".") // if no ext, remove last '.'

		if prevpath == "" {
			prevpath = fullpath
		}
		io.MustWriteFile(prevpath, []byte(value))
		os.Rename(prevpath, fullpath)
	}
}

func (opt *SaveOpt) fileFetch(key string) (string, bool) {
	if opt.Dir != "" {
		absdir, _ := io.AbsPath(opt.Dir, false)
		fullpath := filepath.Join(absdir, key)
		ext := strings.TrimLeft(opt.Ext, ".")

		// search path with .../key(number).ext
		if matches, err := filepath.Glob(fullpath + "(*)." + ext); err == nil {
			if len(matches) > 0 {
				fullpath = matches[0]
			}
		}

		if io.FileExists(fullpath) {
			if bytes, err := os.ReadFile(fullpath); err == nil {
				return string(bytes), true
			}
		}
	}
	return "", false
}

// ----------------------- //

func (opt *SaveOpt) sm(key, value string) {
	if opt.SM != nil {
		opt.SM.Store(key, value)
	}
}

func (opt *SaveOpt) smFetch(key string) (string, bool) {
	if opt.SM != nil {
		if value, ok := opt.SM.Load(key); ok {
			return value.(string), ok
		}
	}
	return "", false
}

// ----------------------- //

func (opt *SaveOpt) m(key, value string) {
	if opt.M != nil {
		opt.M[key] = value
	}
}

func (opt *SaveOpt) mFetch(key string) (string, bool) {
	if opt.M != nil {
		if value, ok := opt.M[key]; ok {
			return value.(string), ok
		}
	}
	return "", false
}

// more save / get func ...

// ----------------------- //

func (opt *SaveOpt) batchSave(key, value string) {

	defer func() {
		if opt.WG != nil {
			opt.WG.Done()
		}
		if opt.Mtx != nil {
			opt.Mtx.Unlock()
		}
	}()
	// work adds first, then mutex lock !
	if opt.WG != nil {
		opt.WG.Add(1)
	}
	if opt.Mtx != nil {
		opt.Mtx.Lock()
	}

	// no key, no saving
	if key == "" {
		return
	}

	if opt.OnFileConflict != nil {
		if str, ok := opt.fileFetch(key); ok { // conflicts
			if save, content := opt.OnFileConflict(str, value); save {
				opt.file(key, content)
			}
			goto SM
		}
	}
	opt.file(key, value)

SM:
	if opt.OnSMapConflict != nil {
		if str, ok := opt.smFetch(key); ok { // conflicts
			if save, content := opt.OnSMapConflict(str, value); save {
				opt.sm(key, content)
			}
			goto M
		}
	}
	opt.sm(key, value)

M:
	if opt.OnMapConflict != nil {
		if str, ok := opt.mFetch(key); ok { // conflicts
			if save, content := opt.OnMapConflict(str, value); save {
				opt.m(key, content)
			}
			goto NEXT
		}
	}
	opt.m(key, value)

	// ... more
NEXT:
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

func (opt *SaveOpt) Factory4IdxSave(start int) func(value string) {
	idx := int64(start - 1)
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
