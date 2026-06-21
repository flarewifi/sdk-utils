package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	tools "core/utils"
	"core/utils/plugins"

	sdkutils "github.com/flarewifi/sdk-utils"
)

func main() {
	// sqlc (db/queries) and templ (*_templ.go) outputs are committed to the
	// repo. On-device core_arch_bin builds run where these tools are unavailable
	// (and sqlc cannot even compile on 32-bit targets), so pass --skip-sqlc and
	// --skip-templ to build from the committed generated files instead.
	skipSqlc := flag.Bool("skip-sqlc", false, "Skip sqlc generation; use the committed db/queries package")
	skipTempl := flag.Bool("skip-templ", false, "Skip templ generation; use the committed *_templ.go files")
	// emit-deps: after building, write the core's external dependency closure
	// (core + sdk/api + sdk/utils, with go.sum hashes) as JSON. The cloud builder
	// reports this so the server seeds the per-core-version plugin dependency lock.
	emitDeps := flag.String("emit-deps", "", "Write the core's resolved dependency closure to this JSON file")
	flag.Parse()

	tools.BuildCoreBins(plugins.BuildOpts{
		SkipTemplates: *skipTempl,
		SkipQueries:   *skipSqlc,
	})

	if *emitDeps != "" {
		if err := emitCoreDeps(*emitDeps); err != nil {
			panic(fmt.Errorf("Error writing core deps to %s: %s", *emitDeps, err.Error()))
		}
	}
}

// emitCoreDeps resolves the core's dependency closure from the current workspace
// (cwd is the build root, holding core/ + sdk/) and writes it as JSON.
func emitCoreDeps(outFile string) error {
	resolved, err := plugins.ResolvedCoreModules(".")
	if err != nil {
		return err
	}
	data, err := json.Marshal(resolved)
	if err != nil {
		return err
	}
	return os.WriteFile(outFile, data, sdkutils.PermFile)
}
