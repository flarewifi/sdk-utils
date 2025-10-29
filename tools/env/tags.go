package env

import "strings"

var (
	BuildTags string = ""
)

func HasGoTag(tag string) bool {
	return strings.Contains(BuildTags, tag)
}
