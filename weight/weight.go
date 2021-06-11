package weight

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	jt "github.com/cdutwhu/json-tool"
	"github.com/nsip/otf-weighting/store"
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

func wtInfo(sid, dom, dt string, score int, others ...string) string {
	if len(others) > 0 {
		return fmt.Sprintf(`{"studentID":"%s","domain":"%s","date":"%s","weightScore":%d,"refer":%s}`, sid, dom, dt, score, others[0])
	}
	return fmt.Sprintf(`{"studentID":"%s","domain":"%s","date":"%s","weightScore":%d}`, sid, dom, dt, score)
}

type RstWt struct {
	Info string
	Err  error
}

// Without Time Factor
// func Process(chRstWt chan<- RstWt, wg *sync.WaitGroup, opt *store.Option, sid, domainpath, scorepath string) {

// 	defer wg.Done()

// 	value, ok := opt.M[sid]
// 	if !ok {
// 		if value, ok = opt.SM.Load(sid); !ok {
// 			chRstWt <- RstWt{Err: fmt.Errorf("sid@%s is not in map or sync.map storage", sid)}
// 			return
// 		}
// 	}

// 	mDomain := make(map[string]string)
// 	chRstObj, _ := jt.ScanObject(strings.NewReader(value.(string)), false, false, jt.OUT_ORI)
// 	for rst := range chRstObj {
// 		domain := gjson.Get(rst.Obj, domainpath).String()
// 		mDomain[domain] = util.PushJA(mDomain[domain], rst.Obj)
// 	}

// 	for domain, otfstr := range mDomain {
// 		score, n := 0, 0
// 		chRstObj, _ := jt.ScanObject(strings.NewReader(otfstr), false, false, jt.OUT_ORI)
// 		for rst := range chRstObj {
// 			score += int(gjson.Get(rst.Obj, scorepath).Int())
// 			n++
// 		}
// 		score = score / n
// 		chRstWt <- RstWt{WtInfo: fmt.Sprintf(`{"studentID":"%s","domain":"%s","weight":%d,"refer":%s}`, sid, domain, score, otfstr)}
// 	}
// }

// With Time Factor
func Process(chRstWt chan<- RstWt, wg *sync.WaitGroup, opt *store.Option, sid, domainpath, timepath, scorepath string) {

	defer wg.Done()

	value, ok := opt.M[sid]
	if !ok {
		if value, ok = opt.SM.Load(sid); !ok {
			chRstWt <- RstWt{Err: fmt.Errorf("sid@%s is not in map or sync.map storage", sid)}
			return
		}
	}

	var (
		mDomDtOTF   = make(map[string]string)
		chRstObj, _ = jt.ScanObject(strings.NewReader(value.(string)), false, false, jt.OUT_ORI)
	)

	for rst := range chRstObj {
		var (
			domdt = gjson.GetMany(rst.Obj, domainpath, timepath)
			dom   = domdt[0].String()
			utc   = domdt[1].String()
		)
		if dom != "" && utc != "" {
			dt, _ := utc2dtm(utc, "yM", "")
			key := fmt.Sprintf("%s@%s", dom, dt)
			mDomDtOTF[key] = util.PushJA(mDomDtOTF[key], rst.Obj)
		}
	}

	for domdt, otfrst := range mDomDtOTF {

		var (
			ss          = strings.Split(domdt, "@")
			dom, dt     = ss[0], ss[1]
			wtScore, n  = 0, 0
			err         error
			chRstObj, _ = jt.ScanObject(strings.NewReader(otfrst), false, false, jt.OUT_ORI)
		)

		for rst := range chRstObj {
			scorerst := gjson.Get(rst.Obj, scorepath)
			score := int(scorerst.Int())
			if score == 0 && scorerst.String() == "" {
				err = fmt.Errorf("OTF..<scaledScore> missing@ Student(%s)-Domain(%s)-Date(%s)", sid, dom, dt)
				if foe, _ := strconv.ParseBool(os.Getenv("FatalOnErr")); foe {
					log.Fatalln(err) // debug, alert to let benthos to remove invalid
				}
				continue
			}
			wtScore += score
			n++
		}

		if n != 0 {
			wtScore = wtScore / n
			chRstWt <- RstWt{Info: wtInfo(sid, dom, dt, wtScore, otfrst), Err: err}
		}
	}
}

func AsyncProc(sidGrp []string, opt *store.Option, domainpath, timepath, scorepath string) <-chan RstWt {

	chRstWt := make(chan RstWt, len(sidGrp))
	wg := &sync.WaitGroup{}
	wg.Add(len(sidGrp))
	for _, sid := range sidGrp {
		// go Process(chRstWt, wg, opt, sid, domainpath, scorepath)
		go Process(chRstWt, wg, opt, sid, domainpath, timepath, scorepath)
	}
	go func() {
		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		close(chRstWt)
	}()
	return chRstWt
}
