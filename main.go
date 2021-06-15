package main

import (
	"fmt"
	"net/http"
	"strconv"

	store "github.com/digisan/data-block/local-kv"
	"github.com/digisan/gotk/slice/ts"
	jt "github.com/digisan/json-tool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsip/otf-weighting/config"
	"github.com/nsip/otf-weighting/log"
	"github.com/nsip/otf-weighting/util"
	"github.com/nsip/otf-weighting/weight"
	"github.com/tidwall/gjson"
)

func main() {

	var (
		cfg        = config.GetConfig("./config/config.toml", "./config.toml")
		mustarray  = cfg.MustInArray
		sidpath    = cfg.Weighting.StudentIDPath
		domainpath = cfg.Weighting.DomainPath
		timepath   = cfg.Weighting.TimePath
		scorepath  = cfg.Weighting.ScorePath

		optLocal   = store.NewOption(cfg.In, cfg.InType, util.Fac4AppendJA, true, true)
		optIn      = optLocal
		optOut     = store.NewOption(cfg.Out, cfg.OutType, util.Fac4AppendJA, false, false)
		optOutSave = optOut.Fac4SaveWithIdxKey(0)

		ilog = log.Factory4IdxLog(0)
	)

	ilog("starting...")
	ilog(fmt.Sprintf("synchronised sid count: %d", optIn.FileSyncToMap()))
	ilog("existing sid count:", len(optIn.M))

	// ------------------------------------------- //

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	// POST eg. /weight?refprev=true
	e.POST(cfg.Service.API, func(c echo.Context) error {

		var (
			refprev      = c.QueryParam("refprev")
			chRstObj, ok = jt.ScanObject(c.Request().Body, mustarray, true, jt.OUT_MIN)
		)

		if rp, err := strconv.ParseBool(refprev); err == nil && !rp {
			dir := util.MakeTempDir(cfg.InTemp)
			optIn = store.NewOption(dir, cfg.InType, util.Fac4AppendJA, true, true)
			defer optLocal.AppendJSONFromFile(dir)
		}
		optInSave := optIn.Save // optIn.Factory4SaveKeyAsIdx(0) SaveKeyAsTS SaveKeyAsID

		switch {
		case !ok && mustarray:
			return echo.NewHTTPError(http.StatusBadRequest, "Not JSON Array")
		case !ok:
			ilog("Single JSON Object")
		}

		// store once POST student ID group
		sidGrp := []string{}
		for rst := range chRstObj {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}
			sid := gjson.Get(rst.Obj, sidpath).String() // fetch sid from studentID path
			sidGrp = append(sidGrp, sid)                // store each sid
			go optInSave(sid, rst.Obj, true)            // save each otf processed json
		}

		// wait for storing finish
		optIn.Wait()

		// process each sid's score weighting
		wtOutput := ""
		chRstWt := weight.AsyncProc(ts.MkSet(sidGrp...), optIn, domainpath, timepath, scorepath)
		for rst := range chRstWt {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, rst.Err.Error())
			}
			wtOutput = util.PushJA(wtOutput, rst.Info)
		}
		optOutSave(wtOutput)

		return c.String(http.StatusOK, wtOutput)
	})

	// GET eg. /weight?sid=12345&domain=math&date=20210607
	e.GET(cfg.Service.API, func(c echo.Context) error {

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

		optAudit := store.NewOption("./audit", "json", util.Fac4AppendJA, false, false)
		optAudit.Clear(true)
		optAudit.Fac4SaveWithIdxKey(0)(wtOut)

		return c.String(http.StatusOK, wtOut)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.Service.Port)))
}
