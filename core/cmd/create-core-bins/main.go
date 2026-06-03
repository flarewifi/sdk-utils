package main

import (
	"flag"

	tools "core/utils"
	"core/utils/plugins"
)

func main() {
	// sqlc (db/queries) and templ (*_templ.go) outputs are committed to the
	// repo. On-device core_arch_bin builds run where these tools are unavailable
	// (and sqlc cannot even compile on 32-bit targets), so pass --skip-sqlc and
	// --skip-templ to build from the committed generated files instead.
	skipSqlc := flag.Bool("skip-sqlc", false, "Skip sqlc generation; use the committed db/queries package")
	skipTempl := flag.Bool("skip-templ", false, "Skip templ generation; use the committed *_templ.go files")
	flag.Parse()

	tools.BuildCoreBins(plugins.BuildOpts{
		SkipTemplates: *skipTempl,
		SkipQueries:   *skipSqlc,
	})
}
