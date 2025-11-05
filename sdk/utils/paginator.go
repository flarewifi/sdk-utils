package sdkutils

import (
	"net/url"
)

func GetPaginationValues(params url.Values) (int, int) {
	var (
		defaultPage    = 1
		defaultPerPage = 10
	)

	page := AtoiOrDefault(params.Get("page"), defaultPage)
	perPage := AtoiOrDefault(params.Get("per_page"), defaultPerPage)

	return page, perPage
}
