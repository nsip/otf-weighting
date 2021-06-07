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

type RstWt struct {
	WtInfo string
	Err    error
}

func Process(chRstWt chan<- RstWt, wg *sync.WaitGroup, opt *store.Option, sid, domainpath, scorepath string) {

	defer wg.Done()

	value, ok := opt.M[sid]
	if !ok {
		if value, ok = opt.SM.Load(sid); !ok {
			chRstWt <- RstWt{Err: fmt.Errorf("sid@%s is not in map or sync.map storage", sid)}
			return
		}
	}

	mDomain := make(map[string]string)
	chRstObj, _ := jt.ScanObject(strings.NewReader(value.(string)), false, false, jt.OUT_ORI)
	for rst := range chRstObj {
		domain := gjson.Get(rst.Obj, domainpath).String()
		mDomain[domain] = util.PushJA(mDomain[domain], rst.Obj)
	}

	for domain, otfstr := range mDomain {
		score, n := 0, 0
		chRstObj, _ := jt.ScanObject(strings.NewReader(otfstr), false, false, jt.OUT_ORI)
		for rst := range chRstObj {
			score += int(gjson.Get(rst.Obj, scorepath).Int())
			n++
		}
		score = score / n
		chRstWt <- RstWt{WtInfo: fmt.Sprintf(`{"studentID":"%s","domain":"%s","weight":%d,"refer":%s}`, sid, domain, score, otfstr)}
	}
}

func AsyncProc(sidGrp []string, opt *store.Option, domainpath, scorepath string) <-chan RstWt {

	chRstWt := make(chan RstWt, len(sidGrp))
	wg := &sync.WaitGroup{}
	wg.Add(len(sidGrp))
	for _, sid := range sidGrp {
		go Process(chRstWt, wg, opt, sid, domainpath, scorepath)
	}
	go func() {
		wg.Wait()
		time.Sleep(10 * time.Millisecond)
		close(chRstWt)
	}()
	return chRstWt
}
