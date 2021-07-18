package weight

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	store "github.com/digisan/data-block/local-kv"
	jt "github.com/digisan/json-tool"
	"github.com/nsip/otf-weighting/util"
	"github.com/tidwall/gjson"
)

// yMd : ["yMd", "yM", "y"]
// hms : ["hms", "hm", "h"]
func utc2dtm(utc, yMd, hms string) (dt, tm string) {
	y, M, d, h, m, s := 0, 0, 0, 0, 0, 0
	fmt.Sscanf(utc, "%04d-%02d-%02dT%02d:%02d:%02dZ", &y, &M, &d, &h, &m, &s)

	switch yMd {
	case "yMd":
		dt = fmt.Sprintf("%04d%02d%02d", y, M, d)
	case "yM":
		dt = fmt.Sprintf("%04d%02d", y, M)
	case "y":
		dt = fmt.Sprintf("%04d", y)
	default:
		dt = fmt.Sprintf("%04d%02d%02d", y, M, d)
	}

	switch hms {
	case "hms":
		tm = fmt.Sprintf("%02d%02d%02d", h, m, s)
	case "hm":
		tm = fmt.Sprintf("%02d%02d", h, m)
	case "h":
		tm = fmt.Sprintf("%02d", h)
	default:
		tm = fmt.Sprintf("%02d%02d%02d", h, m, s)
	}

	return
}

func wtInfo(sid, pl, dt string, score int, others ...string) string {
	if len(others) > 0 {
		return fmt.Sprintf(`{"studentID":"%s","progressionLevel":"%s","date":"%s","weightScore":%d,"refer":%s}`, sid, pl, dt, score, others[0])
	}
	return fmt.Sprintf(`{"studentID":"%s","progressionLevel":"%s","date":"%s","weightScore":%d}`, sid, pl, dt, score)
}

type RstWt struct {
	Info string
	Err  error
}

// With Time Factor
func Process(chRstWt chan<- RstWt, wg *sync.WaitGroup, opt *store.Option, sid, proglvlpath, timepath0, timepath1, scorepath string) {

	defer wg.Done()

	value, ok := opt.M[sid]
	if !ok {
		if value, ok = opt.SM.Load(sid); !ok {
			chRstWt <- RstWt{Err: fmt.Errorf("sid@%s is not in map or sync.map storage", sid)}
			return
		}
	}

	var (
		FatalOnErr, _ = strconv.ParseBool(os.Getenv("FatalOnErr"))
		mProgLvlDtOTF = make(map[string]string)
		chRstObj, _   = jt.ScanObject(strings.NewReader(value.(string)), false, false, jt.OUT_ORI)
	)

	for rst := range chRstObj {
		var (
			pldt = gjson.GetMany(rst.Obj, proglvlpath, timepath0, timepath1)
			pl   = pldt[0].String()
			utc  = pldt[1].String()
			utc1 = pldt[2].String()
		)

		for i, c := range pl {
			if c >= '0' && c <= '9' {
				pl = pl[:i]
				break
			}
		}

		if utc == "" {
			utc = utc1
		}

		if pl != "" && utc != "" {
			dt, _ := utc2dtm(utc, "yM", "")
			key := fmt.Sprintf("%s@%s", pl, dt)
			mProgLvlDtOTF[key] = util.PushJA(mProgLvlDtOTF[key], rst.Obj)
		}
	}

	for proglvl, otfrst := range mProgLvlDtOTF {

		var (
			ss               = strings.Split(proglvl, "@")
			pl, dt           = ss[0], ss[1]
			wtScore, n       = 0, 0
			err              error
			chRstObj4Each, _ = jt.ScanObject(strings.NewReader(otfrst), false, false, jt.OUT_ORI)
		)

		for rst := range chRstObj4Each {
			scorerst := gjson.Get(rst.Obj, scorepath)
			score := int(scorerst.Int())
			if score == 0 && scorerst.String() == "" {
				err = fmt.Errorf("OTF..<scaledScore> missing@ Student(%s)-ProgressionLevel(%s)-Date(%s)", sid, pl, dt)
				if FatalOnErr {
					log.Fatalln(err) // debug, alert to let benthos to remove invalid
				}
				continue
			}
			wtScore += score
			n++
		}

		if n != 0 {
			wtScore = wtScore / n
			chRstWt <- RstWt{Info: wtInfo(sid, pl, dt, wtScore, otfrst), Err: err}
		}
	}
}

func AsyncProc(sidGrp []string, opt *store.Option, proglvlpath, timepath0, timepath1, scorepath string) <-chan RstWt {

	chRstWt := make(chan RstWt, len(sidGrp))
	wg := &sync.WaitGroup{}
	wg.Add(len(sidGrp))
	for _, sid := range sidGrp {
		go Process(chRstWt, wg, opt, sid, proglvlpath, timepath0, timepath1, scorepath)
	}
	go func() {
		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		close(chRstWt)
	}()
	return chRstWt
}
