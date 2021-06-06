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
		ext        = cfg.InboundFileType
		auditdir   = cfg.AuditDir
		port       = cfg.Service.Port
		weightAPI  = cfg.Service.API
		sidpath    = cfg.Weighting.StudentIDPath
		domainpath = cfg.Weighting.DomainPath
		scorepath  = cfg.Weighting.ScorePath

		opt = &store.Option{
			WG:             &sync.WaitGroup{},
			Mtx:            &sync.Mutex{},
			Dir:            auditdir,
			Ext:            ext,
			OnFileConflict: util.FactoryAppendJA(),
			SM:             &sync.Map{},
			OnSMapConflict: util.FactoryAppendJA(),
			M:              map[interface{}]interface{}{},
			OnMapConflict:  util.FactoryAppendJA(),
		}
		save = opt.Save // opt.Factory4SaveKeyAsIdx(0) SaveKeyAsTS SaveKeyAsID

		ilog = log.Factory4IdxLog(0)
	)

	ilog("starting...")

	// ------------------------------------------- //

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	e.POST(weightAPI, func(c echo.Context) error {

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
			go save(sid, rst.Obj)                       // save each otf processed json
		}

		// wait for storing finish
		opt.Wait()

		// process each sid's score weighting
		wtOutput := ""
		chRstWt := weight.AsyncProc(ts.MkSet(sidGrp...), opt, domainpath, scorepath)
		for rst := range chRstWt {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Storage Inconsistent @", rst.Err)
			}
			wtOutput = util.PushJA(wtOutput, rst.WtInfo)
		}

		return c.String(http.StatusOK, wtOutput)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}
