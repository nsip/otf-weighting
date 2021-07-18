package main

import (
	"fmt"
	"os"
	"testing"

	jt "github.com/digisan/json-tool"
)

func TestMain(t *testing.T) {
	main()
}

func TestScan(t *testing.T) {
	r, err := os.Open("./test.json")
	if err == nil {
		ch, _ := jt.ScanObject(r, false, true, jt.OUT_FMT)
		for o := range ch {
			fmt.Println(o.Obj)
		}
	}
}
