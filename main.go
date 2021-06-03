package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	jt "github.com/cdutwhu/json-tool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsip/otf-weighting/store"
)

var (
	port = 1329
)

func main() {

	const keypath = "otf.align.alignmentServiceID"

	var (
		mustarray = false
		opt       = store.SaveOpt{
			WG:  &sync.WaitGroup{},
			Dir: "audit_otf",
			Ext: "json",
			SM:  &sync.Map{},
			M:   nil,
		}
		save = opt.IDXSaveFactory(0) // opt.TSSave // opt.GUIDSave
	)

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	e.POST("/post", func(c echo.Context) error {

		chRst, ok := jt.ScanObject(c.Request().Body, mustarray, true, jt.OUT_MIN)

		if !ok {
			log.Println("Invalid JSON array")
			if mustarray {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON array")
			}
		}

		for rst := range chRst {

			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}

			// key := gjson.Get(rst.Obj, keypath)
			go save(rst.Obj)
		}

		opt.Wait()

		value, _ := opt.SM.Load("NiJ3XA2rB09PvBzhjqNNDS")
		return c.String(http.StatusOK, fmt.Sprint(value))
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}
