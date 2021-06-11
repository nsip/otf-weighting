package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	jt "github.com/cdutwhu/json-tool"
	"github.com/digisan/gotk/io"
	"github.com/digisan/gotk/slice/ts"
	"github.com/google/uuid"
	"github.com/nsip/otf-weighting/util"
)

type (
	Option struct {
		WG  *sync.WaitGroup
		Mtx *sync.Mutex

		Dir, Ext       string                                       // file directory & file extension
		OnFileConflict func(existing, coming string) (bool, string) // file conflict solver

		M             map[interface{}]interface{}                  // map
		OnMapConflict func(existing, coming string) (bool, string) // map conflict solver

		SM             *sync.Map                                    // sync map ptr
		OnSMapConflict func(existing, coming string) (bool, string) // sync map conflict solver
		// more ...
	}
)

func NewOption(dir, ext string, wantM, wantSM bool) *Option {

	opt := &Option{
		WG:             &sync.WaitGroup{},
		Mtx:            &sync.Mutex{},
		Dir:            dir,
		Ext:            ext,
		OnFileConflict: util.FactoryAppendJA(),
	}

	if wantM {
		opt.M = map[interface{}]interface{}{}
		opt.OnMapConflict = util.FactoryAppendJA()
	}

	if wantSM {
		opt.SM = &sync.Map{}
		opt.OnSMapConflict = util.FactoryAppendJA()
	}

	// more ...

	return opt
}

func (opt *Option) file(key, value string, repeatIdx bool) {
	if opt.Dir != "" {
		absdir, _ := io.AbsPath(opt.Dir, false)
		fullpath := filepath.Join(absdir, key) // full abs file name path without extension
		ext := strings.TrimLeft(opt.Ext, ".")  // extension without prefix '.'
		prevpath := ""

		if repeatIdx {
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

func (opt *Option) fileFetch(key string, repeatIdx bool) (string, bool) {
	if opt.Dir != "" {
		absdir, _ := io.AbsPath(opt.Dir, false)
		fullpath := filepath.Join(absdir, key)
		ext := strings.TrimLeft(opt.Ext, ".")

		if repeatIdx {
			// search path with .../key(number).ext
			if matches, err := filepath.Glob(fullpath + "(*)." + ext); err == nil {
				if len(matches) > 0 {
					fullpath = matches[0]
				}
			}
		} else {
			// add extension
			fullpath = strings.TrimRight(fullpath+"."+ext, ".") // if no ext, remove last '.'
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

func (opt *Option) sm(key, value string) {
	if opt.SM != nil {
		opt.SM.Store(key, value)
	}
}

func (opt *Option) smFetch(key string) (string, bool) {
	if opt.SM != nil {
		if value, ok := opt.SM.Load(key); ok {
			return value.(string), ok
		}
	}
	return "", false
}

// ----------------------- //

func (opt *Option) m(key, value string) {
	if opt.M != nil {
		opt.M[key] = value
	}
}

func (opt *Option) mFetch(key string) (string, bool) {
	if opt.M != nil {
		if value, ok := opt.M[key]; ok {
			return value.(string), ok
		}
	}
	return "", false
}

// more save / get func ...

// ----------------------- //

func (opt *Option) batchSave(key, value string, repeatIdx bool) {

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
		if str, ok := opt.fileFetch(key, repeatIdx); ok { // conflicts
			if save, content := opt.OnFileConflict(str, value); save {
				opt.file(key, content, repeatIdx)
			}
			goto M
		}
	}
	opt.file(key, value, repeatIdx)

M:
	if opt.OnMapConflict != nil {
		if str, ok := opt.mFetch(key); ok { // conflicts
			if save, content := opt.OnMapConflict(str, value); save {
				opt.m(key, content)
			}
			goto SM
		}
	}
	opt.m(key, value)

SM:
	if opt.OnSMapConflict != nil {
		if str, ok := opt.smFetch(key); ok { // conflicts
			if save, content := opt.OnSMapConflict(str, value); save {
				opt.sm(key, content)
			}
			goto NEXT
		}
	}
	opt.sm(key, value)

	// ... more
NEXT:
}

func (opt *Option) Wait() {
	if opt.WG != nil {
		opt.WG.Wait()
	}
}

///////////////////////////////////////////////////////

func (opt *Option) Save(key, value string, fileNameRepeatIdx bool) {
	opt.batchSave(key, value, fileNameRepeatIdx)
}

func (opt *Option) Factory4SaveKeyAsIdx(start int) func(value string) {
	idx := int64(start - 1)
	return func(value string) {
		opt.batchSave(fmt.Sprintf("%04d", atomic.AddInt64(&idx, 1)), value, false)
	}
}

func (opt *Option) SaveKeyAsTS(value string) {
	opt.batchSave(time.Now().Format("2006-01-02 15:04:05.000000"), value, false)

	// current := time.Now()
	// // StampMicro
	// fmt.Println("yyyy-mm-dd HH:mm:ss: ", current.Format("2006-01-02 15:04:05.000000"))
	// // yyyy-mm-dd HH:mm:ss:  2016-09-02 15:53:07.159994
	// // StampNano
	// fmt.Println("yyyy-mm-dd HH:mm:ss: ", current.Format("2006-01-02 15:04:05.000000000"))
	// // yyyy-mm-dd HH:mm:ss:  2016-09-02 15:53:07.159994437
}

func (opt *Option) SaveKeyAsID(value string) {
	opt.batchSave(strings.ReplaceAll(uuid.New().String(), "-", ""), value, false)
}

///////////////////////////////////////////////////////

func withDot(str string) string {
	return "." + strings.TrimLeft(str, ".")
}

func (opt *Option) FileSyncToMap() int {
	files, _, err := io.WalkFileDir(opt.Dir, true)
	if err != nil {
		return 0
	}
	return len(ts.FM(
		files,
		func(i int, e string) bool { return strings.HasSuffix(e, withDot(opt.Ext)) },
		func(i int, e string) string {
			fname := filepath.Base(e)
			key := fname[:strings.IndexAny(fname, "(.")]
			if bytes, err := os.ReadFile(e); err == nil {
				value := string(bytes)
				opt.m(key, value)
				opt.sm(key, value)
			} else {
				log.Fatalln(err)
			}
			return key
		},
	))
}

func (opt *Option) AppendJSONFromFile(dir string) int {
	files, _, err := io.WalkFileDir(dir, true)
	if err != nil {
		return 0
	}
	return len(ts.FM(
		files,
		func(i int, e string) bool { return strings.HasSuffix(e, withDot("json")) },
		func(i int, e string) string {
			fname := filepath.Base(e)
			key := fname[:strings.IndexAny(fname, "(.")]
			file, err := os.OpenFile(e, os.O_RDONLY, os.ModePerm)
			if err != nil {
				log.Fatalln(err)
			}
			defer file.Close()

			ResultOfScan, _ := jt.ScanObject(file, false, true, jt.OUT_MIN)
			for rst := range ResultOfScan {
				if rst.Err == nil {
					opt.Save(key, rst.Obj, true)
				}
			}

			return key
		},
	))
}

///////////////////////////////////////////////////////

func (opt *Option) Clear(rmPersistent bool) {
	if rmPersistent {
		if io.DirExists(opt.Dir) {
			os.RemoveAll(opt.Dir)
		}
		// more ...
	}
	if opt.M != nil {
		opt.M = make(map[interface{}]interface{})
	}
	if opt.SM != nil {
		opt.SM = &sync.Map{}
	}
}
