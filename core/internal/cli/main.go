package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"core/internal/cli/server"
	"core/internal/plugindeps"
	tools "core/utils"
	"core/utils/env"
	"core/utils/plugins"

	sdkutils "github.com/flarehotspot/sdk-utils"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println(Usage())
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "env":
		fmt.Println(GoEnvToString(env.GO_ENV))
		return

	case "server":
		server.Server()
		return

	case "create-plugin":
		CreatePlugin()
		return

	case "create-migration":
		CreateMigration()
		return

	case "build-plugin":
		BuildPlugin()
		return

	case "build-plugins":
		BuildPlugins()
		return

	case "build-templates":
		tools.BuildTemplates()
		return

	case "build-queries":
		tools.BuildQueries()
		return

	case "fix-workspace":
		tools.CreateGoWorkspace()
		return

	// case "upgrade":
	// 	updates.UpgradeCmd()
	// 	return

	case "help":
		fmt.Println(Usage())
		return

	case "-h":
		fmt.Println(Usage())
		return

	default:
		fmt.Println("Unrecognized command: " + command)
	}

	fmt.Println(Usage())
	os.Exit(1)
}

func CreatePlugin() {
	var (
		err        error
		pluginPkg  string
		pluginName string
		pluginDesc string
	)

	var (
		domainSample     = "com.mydomain.myplugin"
		pluginNameSample = "My Plugin"
	)

	for len(strings.Split(pluginPkg, ".")) < 3 {
		pluginPkg, err = tools.AskCmdInput(fmt.Sprintf("Enter the plugin package name, for example \"%s\" (without qoutes): ", domainSample))
		if err != nil {
			panic(err)
		}
		if len(strings.Split(pluginPkg, ".")) < 3 {
			fmt.Printf("\nError: Package name must be at least 3 segments, for example \"%s\" (without qoutes): ", domainSample)
		}
	}

	pluginName, err = tools.AskCmdInput(fmt.Sprintf("Enter the plugin name, for example \"%s\" (without qoutes): ", pluginNameSample))
	if err != nil {
		panic(err)
	}

	pluginDesc, err = tools.AskCmdInput("Enter the plugin description: ")
	if err != nil {
		panic(err)
	}

	tools.CreatePlugin(pluginPkg, pluginName, pluginDesc)
}

func CreateMigration() {
	pluginDefs := plugins.LocalPluginSrcDefs()
	pluginPkgs := make([]string, len(pluginDefs)+1)
	pluginPkgs[0] = "core"

	for i, def := range pluginDefs {
		info, err := sdkutils.GetPluginInfoFromPath(def.LocalPath)
		if err != nil {
			fmt.Println("Warning: Error getting plugin info:", err)
			continue
		} else {
			pluginPkgs[i+1] = info.Package
		}
	}

	pluginNums := make([]string, len(pluginPkgs))
	for i, pluginPkg := range pluginPkgs {
		pluginNums[i] = fmt.Sprintf("%d. %s", i+1, pluginPkg)
	}

	selectPkgAsk := fmt.Sprintf("\nSelect the plugin to create the migration for:\n%s\n\nEnter the number of the corresponding plugin", strings.Join(pluginNums, "\n"))

	selectPkg, err := tools.AskCmdInput(selectPkgAsk)
	if err != nil {
		panic(err)
	}

	pluginIdx, err := strconv.Atoi(selectPkg)
	if err != nil {
		panic(err)
	}

	if pluginIdx < 1 || pluginIdx > len(pluginPkgs) {
		panic(fmt.Errorf("Invalid plugin number: %d", pluginIdx))
	}

	pluginPkg := pluginPkgs[pluginIdx-1]

	name, err := tools.AskCmdInput("Enter the migration name, e.g. create_users_table")
	if err != nil {
		panic(err)
	}

	var pluginDir string
	if pluginPkg == "core" {
		pluginDir = filepath.Join("core")
	} else {
		pluginDir = filepath.Join("plugins", "local", pluginPkg)
	}

	tools.MigrationCreate(pluginDir, name)
}

func BuildPlugin() {
	// Flags must precede the optional plugin path (Go's flag parser stops at the
	// first non-flag arg), e.g. `flare build-plugin --skip-sqlc --skip-templ <dir>`.
	fs := flag.NewFlagSet("build-plugin", flag.ExitOnError)
	skipSqlc := fs.Bool("skip-sqlc", false, "Skip sqlc generation; use the committed db/queries package")
	skipTempl := fs.Bool("skip-templ", false, "Skip templ generation; use the committed *_templ.go files")
	// pinned-deps: JSON file with the per-core-version dependency lock to build
	// against. emit-deps: JSON file to write the .so's resolved dependency set to,
	// for the server to fold back into the lock. Both are used by the cloud builder.
	pinnedDepsFile := fs.String("pinned-deps", "", "JSON file of the dependency lock to pin the build to")
	emitDepsFile := fs.String("emit-deps", "", "JSON file to write the build's resolved dependencies to")
	// fetch-deps pulls the per-core-version dependency lock from the cloud (the
	// machine-facing FetchPluginDependencies RPC) instead of a file, so an on-device
	// build is ABI-matched without a builder shuttling the lock. core-version selects
	// which lock (empty = the machine's registered core version).
	fetchDeps := fs.Bool("fetch-deps", false, "Fetch the dependency lock from the cloud and pin the build to it")
	coreVersion := fs.String("core-version", "", "Core version whose dependency lock to fetch (with --fetch-deps; empty = machine's registered version)")
	_ = fs.Parse(os.Args[2:])

	pinned, err := loadPinnedDeps(*pinnedDepsFile)
	if err != nil {
		panic(fmt.Errorf("Error reading pinned deps %s: %s", *pinnedDepsFile, err.Error()))
	}
	// An explicit --pinned-deps file wins; otherwise --fetch-deps pulls from the cloud.
	if len(pinned) == 0 && *fetchDeps {
		pinned = plugindeps.Fetch(*coreVersion)
	}

	opts := plugins.BuildOpts{
		SkipTemplates: *skipTempl,
		SkipQueries:   *skipSqlc,
		PinnedDeps:    pinned,
	}

	// No plugin path given: build all local plugins (previous no-arg behavior).
	rest := fs.Args()
	if len(rest) == 0 {
		if err := plugins.BuildLocalPlugins(opts); err != nil {
			fmt.Println("Error building plugin: " + err.Error())
			os.Exit(1)
		}
		return
	}

	searchPath := rest[0]
	pluginPath, err := sdkutils.FindPluginSrc(searchPath)
	if err != nil {
		panic(fmt.Errorf("Error finding plugin source in %s: %s", searchPath, err.Error()))
	}

	if err := plugins.BuildPlugin(pluginPath, opts); err != nil {
		panic(fmt.Errorf("Error building plugin in %s: %s", pluginPath, err.Error()))
	}

	// Report the exact dependency set the .so was compiled against so the server
	// can fold it into the core version's lock (first-writer-wins).
	if *emitDepsFile != "" {
		if err := emitResolvedDeps(pluginPath, *emitDepsFile); err != nil {
			panic(fmt.Errorf("Error writing resolved deps for %s: %s", pluginPath, err.Error()))
		}
	}
}

func loadPinnedDeps(path string) ([]plugins.LockedGoModule, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pinned []plugins.LockedGoModule
	if err := json.Unmarshal(data, &pinned); err != nil {
		return nil, err
	}
	return pinned, nil
}

func emitResolvedDeps(pluginPath, outFile string) error {
	resolved, err := plugins.ResolvedGoModules(pluginPath)
	if err != nil {
		return err
	}
	data, err := json.Marshal(resolved)
	if err != nil {
		return err
	}
	return os.WriteFile(outFile, data, sdkutils.PermFile)
}

func BuildPlugins() {
	// sqlc (db/queries) and templ (*_templ.go) outputs are committed to the
	// repo, so builders running where these tools are unavailable can pass
	// --skip-sqlc / --skip-templ to build from the committed generated files.
	fs := flag.NewFlagSet("build-plugins", flag.ExitOnError)
	skipSqlc := fs.Bool("skip-sqlc", false, "Skip sqlc generation; use the committed db/queries package")
	skipTempl := fs.Bool("skip-templ", false, "Skip templ generation; use the committed *_templ.go files")
	_ = fs.Parse(os.Args[2:])

	opts := plugins.BuildOpts{
		SkipTemplates: *skipTempl,
		SkipQueries:   *skipSqlc,
	}

	if err := plugins.BuildLocalPlugins(opts); err != nil {
		fmt.Println("Error building local plugins: " + err.Error())
		os.Exit(1)
	}

	if err := plugins.BuildSystemPlugins(opts); err != nil {
		fmt.Println("Error building system plugins: " + err.Error())
		os.Exit(1)
	}
}

func GoEnvToString(e int8) string {
	switch e {
	case env.ENV_DEV:
		return "development"
	case env.ENV_PRODUCTION:
		return "production"
	case env.ENV_SANDBOX:
		return "sandbox"
	}
	return "unknown"
}

func Usage() string {
	return `
Usage: flare <command> [options]

list of commands:
    env                                 Print the build environment

    server                              Start the flare server

    create-plugin                       Create a new plugin

    create-migration                    Create a new migration

    build-plugin <plugin path>          Build plugin.so file. If no plugin path is provided, all plugins will be built.

    build-plugins                       Build plugin.so of all local plugins and stage system plugins' installed data tree (system plugins are linked into core/plugin.so, so no standalone .so is built for them).

    build-templates                     Compile templ files to golang.

    build-queries                       Compile sql queries to golang.

    fix-workspace                       Re-generate the go.work file

`
}
