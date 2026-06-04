package main

import (
	"flag"

	"core/utils/plugins"
)

func main() {
	// sqlc (db/queries) and templ (*_templ.go) outputs are committed to the
	// repo, so builders running where these tools are unavailable can pass
	// --skip-sqlc / --skip-templ to build from the committed generated files.
	skipSqlc := flag.Bool("skip-sqlc", false, "Skip sqlc generation; use the committed db/queries package")
	skipTempl := flag.Bool("skip-templ", false, "Skip templ generation; use the committed *_templ.go files")
	flag.Parse()

	if err := plugins.BuildLocalPlugins(plugins.BuildOpts{
		SkipTemplates: *skipTempl,
		SkipQueries:   *skipSqlc,
	}); err != nil {
		panic(err)
	}
}
