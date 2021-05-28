package main

import (
	"fmt"
	"net/http"
	"sync"

	jt "github.com/cdutwhu/json-tool"
	"github.com/labstack/echo/v4"
	"github.com/tidwall/gjson"
)

const keypath = "otf.align.alignmentServiceID"

var sm = &sync.Map{}
var wg = &sync.WaitGroup{}

func cluster(object, keypath string) {
	defer wg.Done()
	key := gjson.Get(object, keypath)
	sm.Store(key.String(), object)
}

func main() {
	e := echo.New()

	e.POST("/", func(c echo.Context) error {
		chRst, ok := jt.ScanArrayObject(c.Request().Body, true, jt.SOT_MIN)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON array")
		}

		objects := []string{}
		for rst := range chRst {
			if rst.Err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON @", rst.Err)
			}
			objects = append(objects, rst.Obj)
		}

		wg.Add(len(objects))
		for _, obj := range objects {
			go cluster(obj, keypath)
		}
		wg.Wait()

		value, _ := sm.Load("NiJ3XA2rB09PvBzhjqNNDS")
		return c.String(http.StatusOK, fmt.Sprint(value))
	})

	e.Logger.Fatal(e.Start(":1323"))
}
