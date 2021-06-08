package main

import (
	"fmt"
	"net/http"
	"sync"

	jt "github.com/cdutwhu/json-tool"
	"github.com/digisan/gotk/slice/ts"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsip/otf-weighting/config"
	"github.com/nsip/otf-weighting/log"
	"github.com/nsip/otf-weighting/store"
	"github.com/nsip/otf-weighting/util"
	"github.com/nsip/otf-weighting/weight"
	"github.com/tidwall/gjson"
)

func main() {

	var (
		cfg        = config.GetConfig("./config/config.toml", "./config.toml")
		mustarray  = cfg.InboundMustArray
		sidpath    = cfg.Weighting.StudentIDPath
		domainpath = cfg.Weighting.DomainPath
		timepath   = cfg.Weighting.TimePath
		scorepath  = cfg.Weighting.ScorePath

		optIn = &store.Option{
			WG:             &sync.WaitGroup{},
			Mtx:            &sync.Mutex{},
			Dir:            cfg.InStorage,
			Ext:            cfg.InboundType,
			OnFileConflict: util.FactoryAppendJA(),
			SM:             &sync.Map{},
			OnSMapConflict: util.FactoryAppendJA(),
			M:              map[interface{}]interface{}{},
			OnMapConflict:  util.FactoryAppendJA(),
		}
		saveIn = optIn.Save // optIn.Factory4SaveKeyAsIdx(0) SaveKeyAsTS SaveKeyAsID

		optOut = &store.Option{
			WG:             &sync.WaitGroup{},
			Mtx:            &sync.Mutex{},
			Dir:            cfg.OutStorage,
			Ext:            cfg.OutboundType,
			OnFileConflict: util.FactoryAppendJA(),
		}
		saveOut = optOut.Factory4SaveKeyAsIdx(0)

		ilog = log.Factory4IdxLog(0)
	)

	ilog("starting...")

	if cfg.Weighting.ReferPrevRecord {
		optIn.Synchronise()
		ilog("existing sid count:", len(optIn.M))
	}

	// ------------------------------------------- //

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	// POST /weight
	e.POST(cfg.Service.API, func(c echo.Context) error {

		chRstObj, ok := jt.ScanObject(c.Request().Body, mustarray, true, jt.OUT_MIN)

		switch {
		case !ok && mustarray:
			return echo.NewHTTPError(http.StatusBadRequest, "Not JSON Array")
		case !ok:
			ilog("Not JSON Array")
		}

		// store once POST student ID group
		sidGrp := []string{}

		for rst := range chRstObj {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}
			sid := gjson.Get(rst.Obj, sidpath).String() // fetch sid from studentID path
			sidGrp = append(sidGrp, sid)                // store each sid
			go saveIn(sid, rst.Obj, true)               // save each otf processed json
		}

		// wait for storing finish
		optIn.Wait()

		// process each sid's score weighting
		wtOutput := ""
		chRstWt := weight.AsyncProc(ts.MkSet(sidGrp...), optIn, domainpath, timepath, scorepath)
		for rst := range chRstWt {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, rst.Err.Error())
			}
			wtOutput = util.PushJA(wtOutput, rst.Info)
		}
		saveOut(wtOutput)

		return c.String(http.StatusOK, wtOutput)
	})

	// GET eg. /weight?sid=12345&domain=math&date=20210607
	e.GET(cfg.Service.API, func(c echo.Context) error {

		optAudit := &store.Option{
			WG:             &sync.WaitGroup{},
			Mtx:            &sync.Mutex{},
			Dir:            "./audit",
			Ext:            "json",
			OnFileConflict: util.FactoryAppendJA(),
		}
		optAudit.Clear()
		saveAudit := optAudit.Factory4SaveKeyAsIdx(0)

		var (
			sid     = c.QueryParam("sid")
			domain  = c.QueryParam("domain")
			date    = c.QueryParam("date")
			wtOut   = ""
			chRstWt = weight.AsyncProc([]string{sid}, optIn, domainpath, timepath, scorepath)
		)

		for rst := range chRstWt {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, rst.Err.Error())
			}

			domValue := gjson.Get(rst.Info, "domain").String()
			dateValue := gjson.Get(rst.Info, "date").String()

			switch {
			case (domValue == domain && dateValue == date) ||
				(domValue == domain && date == "") ||
				(domain == "" && dateValue == date) ||
				(domain == "" && date == ""):
				wtOut = util.PushJA(wtOut, rst.Info)
			}
		}
		saveAudit(wtOut)

		return c.String(http.StatusOK, wtOut)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.Service.Port)))
}
