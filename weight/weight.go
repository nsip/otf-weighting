package weight

import (
	"fmt"
	"strings"
	"sync"
	"time"

	jt "github.com/cdutwhu/json-tool"
	"github.com/nsip/otf-weighting/store"
	"github.com/nsip/otf-weighting/util"
	"github.com/tidwall/gjson"
)

func utc2dtm(utc string) (dt, tm string) {
	y, M, d, h, m, s := 0, 0, 0, 0, 0, 0
	fmt.Sscanf(utc, "%04d-%02d-%02dT%02d:%02d:%02dZ", &y, &M, &d, &h, &m, &s)
	return fmt.Sprintf("%04d%02d%02d", y, M, d), fmt.Sprintf("%02d%02d%02d", h, m, s)
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

func Process(chRstWt chan<- RstWt, wg *sync.WaitGroup, opt *store.Option, sid, domainpath, timepath, scorepath string) {

	defer wg.Done()

	value, ok := opt.M[sid]
	if !ok {
		if value, ok = opt.SM.Load(sid); !ok {
			chRstWt <- RstWt{Err: fmt.Errorf("sid@%s is not in map or sync.map storage", sid)}
			return
		}
	}

	mDomDtOTF := make(map[string]string)

	chRstObj, _ := jt.ScanObject(strings.NewReader(value.(string)), false, false, jt.OUT_ORI)
	for rst := range chRstObj {
		domdt := gjson.GetMany(rst.Obj, domainpath, timepath)
		if dom := domdt[0].String(); dom != "" {
			if utc := domdt[1].String(); utc != "" {
				dt, _ := utc2dtm(utc)
				key := fmt.Sprintf("%s@%s", dom, dt)
				mDomDtOTF[key] = util.PushJA(mDomDtOTF[key], rst.Obj)
			}
		}
	}

	for domdt, otfrst := range mDomDtOTF {
		ss := strings.Split(domdt, "@")
		dom, dt := ss[0], ss[1]
		score, n := 0, 0
		chRstObj, _ := jt.ScanObject(strings.NewReader(otfrst), false, false, jt.OUT_ORI)
		for rst := range chRstObj {
			if gjson.Get(rst.Obj, scorepath).String() == "" {
				chRstWt <- RstWt{Err: fmt.Errorf("OTF..<scaledScore> missing@ Student(%s)-Domain(%s)-Date(%s)", sid, dom, dt)}
			}
			score += int(gjson.Get(rst.Obj, scorepath).Int())
			n++
		}
		score = score / n
		chRstWt <- RstWt{Info: wtInfo(sid, dom, dt, score, otfrst)}
	}
}

func AsyncProc(sidGrp []string, opt *store.Option, domainpath, timepath, scorepath string) <-chan RstWt {

	chRstWt := make(chan RstWt, len(sidGrp))
	wg := &sync.WaitGroup{}
	wg.Add(len(sidGrp))
	for _, sid := range sidGrp {
		go Process(chRstWt, wg, opt, sid, domainpath, timepath, scorepath)
		// go Process(chRstWt, wg, opt, sid, domainpath, scorepath)
	}
	go func() {
		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		close(chRstWt)
	}()
	return chRstWt
}
