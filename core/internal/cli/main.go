package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"core/internal/cli/server"
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
	_ = fs.Parse(os.Args[2:])

	opts := plugins.BuildOpts{
		SkipTemplates: *skipTempl,
		SkipQueries:   *skipSqlc,
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
		log.Fatalf("Error finding plugin source in %s: %s\n", searchPath, err.Error())
	}

	if err := plugins.BuildPlugin(pluginPath, opts); err != nil {
		log.Fatalf("Error building plugin in %s: %s\n", pluginPath, err.Error())
	}
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

    build-plugins                       Build plugin.so of all the local and system plugins. Similar to build-plugin command without arguments.

    build-templates                     Compile templ files to golang.

    build-queries                       Compile sql queries to golang.

    fix-workspace                       Re-generate the go.work file

`
}
