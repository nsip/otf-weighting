package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/digisan/data-block/store"
	gotkio "github.com/digisan/gotk/io"
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

		s4in    = store.NewKV(cfg.In, cfg.InType, true, true)
		inSave  = s4in.Save
		s4out   = store.NewKV(cfg.Out, cfg.OutType, true, true)
		outSave = s4out.Fac4SaveWithIdxKey(0)

		ilog     = lk.Fac4GrpIdxLogF("", 1, lk.INFO, false)
		iCntlog  = lk.Fac4GrpIdxLogF("--------------------", 1, lk.INFO, false)
		iCntPost = lk.Fac4GrpIdxLogF("********************", 1, lk.INFO, false)

		mtx = &sync.Mutex{}
	)

	s4in.OnConflict = util.AppendJA
	s4out.OnConflict = util.AppendJA

	lk.Log("starting...")
	lk.Log("synchronised sid count: %d", s4in.FileSyncToMap())
	lk.Log("existing sid count: %d", s4in.KVs[0].Len())

	// same key for multiple test score list
	mGroup := make(map[string][]string)

	// ------------------------------------------- //

	e := echo.New()
	e.Use(middleware.BodyLimit("2G"))

	// POST eg. /weight?refprev=true
	e.POST(cfg.Service.API, func(c echo.Context) error {
		defer mtx.Unlock()
		mtx.Lock()

		iCntPost("--- IN POST ---")

		// time.Sleep(time.Millisecond * 10)
		// return c.String(http.StatusOK, "")

		var (
			key4obj = ""
			_       = c.QueryParam("refprev")
		)

		chRstObj, ok := jt.ScanObject(c.Request().Body, mustarray, true, jt.OUT_MIN)

		// if rp, err := strconv.ParseBool(refprev); err == nil && !rp {
		// 	dir := util.MakeTempDir(cfg.InTemp)
		// 	optIn = store.NewKV(dir, cfg.InType, true, true)
		// 	defer optLocal.AppendJSONFromFile(dir)
		// }

		switch {
		case !ok && mustarray:
			return echo.NewHTTPError(http.StatusBadRequest, "Not JSON Array")
		case !ok:
			ilog("Single JSON Object")
		case ok:
			ilog("JSON Array")
		}

		// store once POST student ID group
		sidGrp := []string{}
		for rst := range chRstObj {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}
			sid := gjson.Get(rst.Obj, sidpath).String() // fetch sid from studentID path
			sidGrp = append(sidGrp, sid)                // store each sid
			inSave(sid, rst.Obj)                        // save each otf processed json

			// make key4obj
			r := gjson.GetMany(rst.Obj, sidpath, proglvlpath, timepath0, timepath1)
			id, pl, dt, dt1 := r[0].String(), r[1].String(), r[2].String(), r[3].String()
			for i, c := range pl {
				if c >= '0' && c <= '9' {
					pl = pl[:i]
					break
				}
			}
			if dt == "" {
				dt = dt1
			}
			dt = dt[:4] + dt[5:7] // dt & dt1 both have '2020-06-02'
			key4obj = fmt.Sprintf("%s#%s#%s", id, pl, dt)
			iCntlog(key4obj)
		}

		// process each sid's score weighting
		for rst := range weight.AsyncProc(ts.MkSet(sidGrp...), s4in, proglvlpath, timepath0, timepath1, scorepath) {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, rst.Err.Error())
			}
			// outSave(rst.Info) // temp check

			//////////////////////////////////////////

			r := gjson.GetMany(rst.Info, "studentID", "progressionLevel", "date")
			id, pl, dt := r[0].String(), r[1].String(), r[2].String()
			key := fmt.Sprintf("%s#%s#%s", id, pl, dt)
			mGroup[key] = append(mGroup[key], rst.Info)
			// iCntlog("%s weighted", key)
		}

		//////////////////////////////////////////
		os.RemoveAll("stat.txt")
		for k, v := range mGroup {
			data := fmt.Sprintf("%s --- %d", k, len(v))
			gotkio.MustAppendFile("stat.txt", []byte(data), true)
		}

		grp := mGroup[key4obj]
		if len(grp) == 0 {
			lk.Warn("ERROR key4obj @ %s", key4obj)
			gotkio.MustAppendFile("panic.txt", []byte(key4obj), true)
			panic("ERROR")
		}

		last := grp[len(grp)-1]
		outSave(last)
		return c.String(http.StatusOK, last)
	})

	// GET eg. /weight?sid=12345&progressionlevel=LWCrT&date=202106
	e.GET(cfg.Service.API, func(c echo.Context) error {

		var (
			sid     = c.QueryParam("sid")
			proglvl = c.QueryParam("progressionlevel")
			date    = c.QueryParam("date")
			wtOut   = ""
		)

		for rst := range weight.AsyncProc([]string{sid}, s4in, proglvlpath, timepath0, timepath1, scorepath) {
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

		optAudit := store.NewKV("./audit", "json", false, false)
		optAudit.Clear(true)
		optAudit.Fac4SaveWithIdxKey(0)(wtOut)

		return c.String(http.StatusOK, wtOut)
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.Service.Port)))
}
