package main

import (
	"fmt"
	"net/http"
	"sync"

	jt "github.com/cdutwhu/json-tool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsip/otf-weighting/log"
	"github.com/nsip/otf-weighting/store"
	"github.com/tidwall/gjson"
)

var (
	port = 1329
)

func main() {

	const keypath = "otf.id.studentID"

	var (
		ilog      = log.Factory4IdxLog(0)
		mustarray = false
		opt       = store.SaveOpt{
			WG:             &sync.WaitGroup{},
			Mtx:            &sync.Mutex{},
			Dir:            "audit_otf",
			Ext:            "json",
			OnFileConflict: FactoryAppendJA(),
			SM:             &sync.Map{},
			OnSMapConflict: FactoryAppendJA(),
			M:              map[interface{}]interface{}{},
			OnMapConflict:  FactoryAppendJA(),
		}
		save = opt.Save // opt.Factory4IdxSave(0) // opt.TSSave // opt.GUIDSave
	)

	ilog("starting...")

	// ------------------------------------------- //

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	e.POST("/post", func(c echo.Context) error {

		chRst, ok := jt.ScanObject(c.Request().Body, mustarray, true, jt.OUT_MIN)

		switch {
		case !ok && mustarray:
			return echo.NewHTTPError(http.StatusBadRequest, "Not JSON Array")
		case !ok:
			ilog("Not JSON Array")
		}

		for rst := range chRst {

			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}

			go save(gjson.Get(rst.Obj, keypath).String(), rst.Obj)
		}

		opt.Wait()

		value, _ := opt.SM.Load("NiJ3XA2rB09PvBzhjqNNDS")
		return c.String(http.StatusOK, fmt.Sprint(value))
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}
