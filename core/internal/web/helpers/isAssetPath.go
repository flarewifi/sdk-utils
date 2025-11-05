package helpers

import (
	"regexp"
)

func IsAssetPath(p string) bool {
	match, err := regexp.MatchString(`\.(js|css|map|png|jpg|jpeg|ico|svg|gif|ttf|woff2?|eot|html|vue|json)$`, p)
	if err != nil {
		return false
	}
	return match
}
