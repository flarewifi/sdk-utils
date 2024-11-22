package sdkstr

import "strings"

func TrimChars(str string, chars ...string) string {
	for _, c := range chars {
		str = strings.Trim(str, c)
	}
	return str
}
