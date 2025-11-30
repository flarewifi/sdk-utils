package tags

import "strings"

func HasGoTag(tag string) bool {
	t := GetBuildTags()
	return strings.Contains(t, tag)
}

func GetBuildTags() string {
	tags := []string{
		env(),
		mono(),
		cgoTag(),
		database(),
	}
	return strings.Join(tags, " ")
}
