package tags

import "strings"

func HasGoTag(tag string) bool {
	t := GetBuildTags()
	return strings.Contains(t, tag)
}

func GetBuildTags() string {
	return env() + " " + mono() + " " + database()
}
