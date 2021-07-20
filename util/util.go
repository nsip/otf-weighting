package util

import (
	"fmt"
	"time"
)

func PushJA(existing, coming string) string {
	if len(existing) > 0 {
		switch existing[0] {
		case '{':
			return fmt.Sprintf("[%s,%s]", existing, coming)
		case '[':
			return fmt.Sprintf("%s,%s]", existing[:len(existing)-1], coming)
		default:
			panic("error in existing JSON storage")
		}
	}
	return coming
}

func AppendJA(existing, coming interface{}) (bool, interface{}) {
	switch existing := existing.(type) {
	case string:
		if len(existing) > 0 {
			switch existing[0] {
			case '{':
				return true, fmt.Sprintf("[%s,%s]", existing, coming)
			case '[':
				return true, fmt.Sprintf("%s,%s]", existing[:len(existing)-1], coming)
			default:
				panic("error in existing JSON storage")
			}
		}
		return true, coming
	default:
		return false, ""
	}
}

func MakeTempDir(dir string) string {
	if dir == "" {
		dir = "temp"
	}
	milsec := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("./%s/%d", dir, milsec)
}
