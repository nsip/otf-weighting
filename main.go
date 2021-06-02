package main

import (
	"fmt"
	"net/http"
	"sync"

	jt "github.com/cdutwhu/json-tool"
	gotkio "github.com/digisan/gotk/io"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tidwall/gjson"
)

var (
	port = 1329
	sm   = &sync.Map{}
	wg   = &sync.WaitGroup{}
)

func factoryStore(dir string) func(object []byte) {
	var fIdx = 0
	return func(object []byte) {
		fIdx++
		gotkio.MustWriteFile(fmt.Sprintf("./%s/%03d.json", dir, fIdx), object)
	}
}

func cluster(object, keypath string) {

	// time.Sleep(1 * time.Second) // delay test

	key := gjson.Get(object, keypath)
	sm.Store(key.String(), object)
	wg.Done()
}

func main() {

	const keypath = "otf.align.alignmentServiceID"

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	e.POST("/", func(c echo.Context) error {

		var (
			store = factoryStore("audit")
		)

		chRst, ok := jt.ScanArrayObject(c.Request().Body, true, jt.OUT_MIN)

		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON array")
		}

		for rst := range chRst {

			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}

			// audit
			go store([]byte(rst.Obj))

			// save to map
			wg.Add(1)
			go cluster(rst.Obj, keypath)
		}

		wg.Wait()

		value, _ := sm.Load("NiJ3XA2rB09PvBzhjqNNDS")
		return c.String(http.StatusOK, fmt.Sprint(value))
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}
