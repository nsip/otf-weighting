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
)

type (
	Fac4Solver func() func(existing, coming interface{}) (bool, interface{})

	Option struct {
		wg  *sync.WaitGroup
		mtx *sync.Mutex

		onConflict [3]func(existing, coming interface{}) (bool, interface{}) // conflict solvers
		dir, ext   string                                                    // file directory & file extension
		M          map[interface{}]interface{}                               // map
		SM         *sync.Map                                                 // sync map ptr
		// more ...
	}
)

func NewOption(dir, ext string, fac4solver Fac4Solver, wantM, wantSM bool) *Option {

	opt := &Option{
		wg:  &sync.WaitGroup{},
		mtx: &sync.Mutex{},
	}

	if dir != "" {
		opt.dir = dir
		opt.ext = ext
		opt.onConflict[0] = fac4solver()
	}

	if wantM {
		opt.M = map[interface{}]interface{}{}
		opt.onConflict[1] = fac4solver()
	}

	if wantSM {
		opt.SM = &sync.Map{}
		opt.onConflict[2] = fac4solver()
	}

	// more ...

	return opt
}

func (opt *Option) file(key, value string, repeatIdx bool) {
	if opt.dir != "" {
		absdir, _ := io.AbsPath(opt.dir, false)
		fullpath := filepath.Join(absdir, key) // full abs file name path without extension
		ext := strings.TrimLeft(opt.ext, ".")  // extension without prefix '.'
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
	if opt.dir != "" {
		absdir, _ := io.AbsPath(opt.dir, false)
		fullpath := filepath.Join(absdir, key)
		ext := strings.TrimLeft(opt.ext, ".")

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

func (opt *Option) m(key, value interface{}) {
	if opt.M != nil {
		opt.M[key] = value
	}
}

func (opt *Option) mFetch(key interface{}) (interface{}, bool) {
	if opt.M != nil {
		if value, ok := opt.M[key]; ok {
			return value, ok
		}
	}
	return nil, false
}

// ----------------------- //

func (opt *Option) sm(key, value interface{}) {
	if opt.SM != nil {
		opt.SM.Store(key, value)
	}
}

func (opt *Option) smFetch(key interface{}) (interface{}, bool) {
	if opt.SM != nil {
		if value, ok := opt.SM.Load(key); ok {
			return value, ok
		}
	}
	return nil, false
}

// more save / get func ...

// ----------------------- //

func (opt *Option) batchSave(key, value string, repeatIdx bool) {

	defer func() {
		if opt.wg != nil {
			opt.wg.Done()
		}
		if opt.mtx != nil {
			opt.mtx.Unlock()
		}
	}()
	// work adds first, then mutex lock !
	if opt.wg != nil {
		opt.wg.Add(1)
	}
	if opt.mtx != nil {
		opt.mtx.Lock()
	}

	// no key, no saving
	if key == "" {
		return
	}

	if solver := opt.onConflict[0]; solver != nil {
		if str, ok := opt.fileFetch(key, repeatIdx); ok { // conflicts
			if save, content := solver(str, value); save {
				opt.file(key, content.(string), repeatIdx)
			}
			goto M
		}
	}
	opt.file(key, value, repeatIdx)

M:
	if solver := opt.onConflict[1]; solver != nil {
		if str, ok := opt.mFetch(key); ok { // conflicts
			if save, content := solver(str, value); save {
				opt.m(key, content)
			}
			goto SM
		}
	}
	opt.m(key, value)

SM:
	if solver := opt.onConflict[2]; solver != nil {
		if str, ok := opt.smFetch(key); ok { // conflicts
			if save, content := solver(str, value); save {
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
	if opt.wg != nil {
		opt.wg.Wait()
	}
}

///////////////////////////////////////////////////////

func (opt *Option) Save(key, value string, fileNameRepeatIdx bool) {
	opt.batchSave(key, value, fileNameRepeatIdx)
}

func (opt *Option) Fac4SaveKeyAsIdx(start int) func(value string) {
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
	files, _, err := io.WalkFileDir(opt.dir, true)
	if err != nil {
		return 0
	}
	return len(ts.FM(
		files,
		func(i int, e string) bool { return strings.HasSuffix(e, withDot(opt.ext)) },
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
		if io.DirExists(opt.dir) {
			os.RemoveAll(opt.dir)
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
