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

// IsDev returns true if running in development mode
func IsDev() bool {
	return env() == "dev"
}

// IsStaging returns true if running in staging mode
func IsStaging() bool {
	return env() == "staging"
}

// IsProd returns true if running in production mode
func IsProd() bool {
	return env() == "prod"
}
