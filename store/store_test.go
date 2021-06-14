package store

import (
	"fmt"
	"testing"

	"github.com/nsip/otf-weighting/util"
)

func TestOption_FileSyncToMap(t *testing.T) {
	opt := NewOption("../in", "json", util.Fac4AppendJA, true, true)
	opt.FileSyncToMap()
	fmt.Println(opt.M["5"])
	fmt.Println(opt.SM.Load("5"))
}

func TestOption_AppendJSONFromFile(t *testing.T) {
	opt := NewOption("../in1", "json", util.Fac4AppendJA, true, true)
	opt.AppendJSONFromFile("../in")
}
