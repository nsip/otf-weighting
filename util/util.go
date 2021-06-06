package util

import "fmt"

func PushJA(existing, coming string) string {
	if len(existing) > 0 {
		switch existing[0] {
		case '{':
			return fmt.Sprintf("[%s,%s]", existing, coming)
		case '[':
			return fmt.Sprintf("%s,%s]", existing[:len(existing)-1], coming)
		default:
			panic("error in existing storage")
		}
	}
	return coming
}

func FactoryAppendJA() func(existing, coming string) (bool, string) {
	return func(existing, coming string) (bool, string) {
		if len(existing) > 0 {
			switch existing[0] {
			case '{':
				return true, fmt.Sprintf("[%s,%s]", existing, coming)
			case '[':
				return true, fmt.Sprintf("%s,%s]", existing[:len(existing)-1], coming)
			default:
				panic("error in existing storage")
			}
		}
		return true, coming
	}
}
