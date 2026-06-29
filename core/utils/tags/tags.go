package tags

import "strings"

func HasGoTag(tag string) bool {
	t := GetBuildTags()
	return strings.Contains(t, tag)
}

func GetBuildTags() string {
	tags := []string{
		env(),
		devkit(),
		mono(),
		cgoTag(),
		database(),
	}
	return strings.Join(tags, " ")
}

// IsDevkit returns true when compiled with the devkit build tag (the
// developer-distribution build). Backed by the devkit()/!devkit build-tag
// variants, so it is resolved at compile time.
func IsDevkit() bool {
	return devkit() == "devkit"
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
