package main

import (
	"fmt"
	"net/http"
	"strconv"

	store "github.com/digisan/data-block/local-kv"
	"github.com/digisan/gotk/slice/ts"
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
		cfg         = config.GetConfig("./config/config.toml", "./config.toml")
		mustarray   = cfg.MustInArray
		sidpath     = cfg.Weighting.StudentIDPath
		proglvlpath = cfg.Weighting.ProgressionLevelPath
		timepath0   = cfg.Weighting.TimePath0
		timepath1   = cfg.Weighting.TimePath1
		scorepath   = cfg.Weighting.ScorePath

		optLocal   = store.NewOption(cfg.In, cfg.InType, util.Fac4AppendJA, true, true)
		optIn      = optLocal
		optOut     = store.NewOption(cfg.Out, cfg.OutType, util.Fac4AppendJA, false, false)
		optOutSave = optOut.Fac4SaveWithIdxKey(0)

		ilog = lk.Fac4GrpIdxLogF("", 0, lk.INFO, false)
	)

	ilog("starting...")
	ilog("synchronised sid count: %d", optIn.FileSyncToMap())
	ilog("existing sid count: %d", len(optIn.M))

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
		chRstWt := weight.AsyncProc(ts.MkSet(sidGrp...), optIn, proglvlpath, timepath0, timepath1, scorepath)
		for rst := range chRstWt {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, rst.Err.Error())
			}
			wtOutput = util.PushJA(wtOutput, rst.Info)
		}
		optOutSave(wtOutput)

		return c.String(http.StatusOK, wtOutput)
	})

	// GET eg. /weight?sid=12345&progressionlevel=LWCrT&date=20210607
	e.GET(cfg.Service.API, func(c echo.Context) error {

		var (
			sid     = c.QueryParam("sid")
			proglvl = c.QueryParam("progressionlevel")
			date    = c.QueryParam("date")
			wtOut   = ""
			chRstWt = weight.AsyncProc([]string{sid}, optIn, proglvlpath, timepath0, timepath1, scorepath)
		)

		for rst := range chRstWt {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, rst.Err.Error())
			}

			plValue := gjson.Get(rst.Info, "progressionLevel").String()
			dateValue := gjson.Get(rst.Info, "date").String()

			switch {
			case (plValue == proglvl && dateValue == date) ||
				(plValue == proglvl && date == "") ||
				(proglvl == "" && dateValue == date) ||
				(proglvl == "" && date == ""):
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
