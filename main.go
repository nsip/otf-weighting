package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/digisan/data-block/store"
	jt "github.com/digisan/json-tool"
	lk "github.com/digisan/logkit"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsip/otf-weighting/config"
	"github.com/nsip/otf-weighting/util"
	"github.com/nsip/otf-weighting/weight"
	"github.com/tidwall/gjson"
)

func main() {

	var (
		cfg       = config.GetConfig("./config/config.toml", "./config.toml")
		mustarray = cfg.MustInArray
		sidpath   = cfg.Weighting.StudentIDPath

		proglvlpath = cfg.Weighting.ProgressionLevelPath
		timepath0   = cfg.Weighting.TimePath0
		timepath1   = cfg.Weighting.TimePath1
		scorepath   = cfg.Weighting.ScorePath

		cSID  = make(chan string, 10000)
		s4in  = store.NewKV(cfg.In, cfg.InType, true, true)
		s4out = store.NewKV(cfg.Out, cfg.OutType, false, false)

		ilog     = lk.Fac4GrpIdxLogF("", 1, lk.INFO, false)
		iCntPost = lk.Fac4GrpIdxLogF("********************", 1, lk.INFO, false)

		mtx = &sync.Mutex{}
	)

	s4in.OnConflict(util.AppendJA)

	// lk.Log("synchronised sid count: %d", s4in.FileSyncToMap())
	// lk.Log("existing sid count: %d", s4in.KVs[0].Len())

	done := make(chan struct{})
	go func() {
		for cnt := range s4in.UnchangedTickerNotifier(2000, true, done, 0) {
			fmt.Println(" --- ", cnt)
			weight.MakeResult(s4in, s4out, cSID, proglvlpath, timepath0, timepath1, scorepath)
		}
	}()

	// -------------------------------------------------------------------------------------- //

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	// POST eg. /weight?refprev=true
	e.POST(cfg.Service.API, func(c echo.Context) error {
		defer mtx.Unlock()
		mtx.Lock()

		iCntPost("--- IN POST ---")

		// time.Sleep(time.Millisecond * 10)
		// return c.String(http.StatusOK, "")

		chRstObj, ok := jt.ScanObject(c.Request().Body, mustarray, true, jt.OUT_MIN)

		switch {
		case !ok && mustarray:
			return echo.NewHTTPError(http.StatusBadRequest, "Not JSON Array")
		case !ok:
			ilog("Single JSON Object")
		case ok:
			ilog("JSON Array")
		}

		sidgrp := []string{}
		for rst := range chRstObj {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}
			sid := gjson.Get(rst.Obj, sidpath).String() // fetch sid from studentID path
			s4in.Save(sid, rst.Obj)                     // save each otf processed json ***
			cSID <- sid                                 // save incoming student id
			sidgrp = append(sidgrp, sid)
		}

		return c.String(http.StatusOK, fmt.Sprintf("%v", sidgrp))
	})

	// GET eg. /weight?sid=12345&progressionlevel=LWCrT&date=202106
	// e.GET(cfg.Service.API, func(c echo.Context) error {

	// 	var (
	// 		sid     = c.QueryParam("sid")
	// 		proglvl = c.QueryParam("progressionlevel")
	// 		date    = c.QueryParam("date")
	// 		wtOut   = ""
	// 	)

	// 	for rst := range weight.AsyncProc([]string{sid}, s4in, proglvlpath, timepath0, timepath1, scorepath) {
	// 		if rst.Err != nil {
	// 			return echo.NewHTTPError(http.StatusBadRequest, rst.Err.Error())
	// 		}

	// 		plValue := gjson.Get(rst.Info, "progressionLevel").String()
	// 		dateValue := gjson.Get(rst.Info, "date").String()

	// 		switch {
	// 		case (plValue == proglvl && dateValue == date) ||
	// 			(plValue == proglvl && date == "") ||
	// 			(proglvl == "" && dateValue == date) ||
	// 			(proglvl == "" && date == ""):
	// 			wtOut = util.PushJA(wtOut, rst.Info)
	// 		}
	// 	}

	// 	optAudit := store.NewKV("./audit", "json", false, false)
	// 	optAudit.Clear(true)
	// 	optAudit.Fac4SaveWithIdxKey(0)(wtOut)

	// 	return c.String(http.StatusOK, wtOut)
	// })

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.Service.Port)))
}
