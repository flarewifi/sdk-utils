package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strconv"
	"strings"
	"time"

	tools "core/build/tools"
	"core/env"
	"core/internal/utils/pkg"
	"core/internal/utils/updates"

	sdkpaths "github.com/flarehotspot/go-utils/paths"
)

var (
	gowork bool
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
		Server()
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
		BuildPlugin()
		return

	case "build-templates":
		tools.BuildTemplates()
		return

	case "fix-workspace":
		tools.CreateGoWorkspace()
		return

	case "install-go":
		var installPath string
		if len(os.Args) > 2 {
			installPath = os.Args[2]
		}
		tools.InstallGo(installPath)
		return

	case "update":
		Update()
		return

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

	for len(strings.Split(pluginPkg, ".")) < 3 {
		pluginPkg, err = tools.AskCmdInput("Enter the plugin package name, e.g. com.mydomain.plugin")
		if err != nil {
			panic(err)
		}
		if len(strings.Split(pluginPkg, ".")) < 3 {
			fmt.Println("Error: Package name must be at least 3 segments. For example: com.my-domain.my-plugin")
		}
	}

	pluginName, err = tools.AskCmdInput("Enter the plugin name, e.g. MyPlugin")
	if err != nil {
		panic(err)
	}

	pluginDesc, err = tools.AskCmdInput("Enter the plugin description")
	if err != nil {
		panic(err)
	}

	tools.CreatePlugin(pluginPkg, pluginName, pluginDesc)
}

func CreateMigration() {
	pluginPaths := pkg.LocalPluginPaths()
	pluginPkgs := make([]string, len(pluginPaths))
	for i, pluginPath := range pluginPaths {
		pluginPkgs[i] = filepath.Base(pluginPath)
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

	pluginDir := filepath.Join("plugins", pluginPkg)
	tools.MigrationCreate(pluginDir, name)
}

func BuildPlugin() {
	var err error
	if len(os.Args) < 3 {
		err = pkg.BuildLocalPlugins()
	} else {
		pluginPath := os.Args[2]
		workdir := filepath.Join(sdkpaths.TmpDir, "builds", filepath.Base(pluginPath))
		err = pkg.BuildPluginSo(pluginPath, workdir)
	}
	if err != nil {
		fmt.Println("Error building plugin: " + err.Error())
		os.Exit(1)
	}
}

func Server() {
	corePath := filepath.Join(sdkpaths.AppDir, "core/plugin.so")
	p, err := plugin.Open(corePath)
	if err != nil {
		log.Println("Error loading core plugin:", err)
		panic(err)
	}
	symInit, _ := p.Lookup("Init")
	initFn := symInit.(func())
	initFn()
}

func Update() {
	fmt.Println("Updating flare system's core..")

	if updates.IsSpawnedFromFlare() {
		fmt.Println("Spawned from flare")
		fmt.Println("killing spawner..")

		// get flare cli pid
		ppid := os.Getppid()
		pproc, err := os.FindProcess(ppid)
		if err != nil {
			log.Println("Error finding parent procces id:", err)
			return
		}

		// stop the flare cli, if running
		if updates.IsProcRunning(pproc) {
			// kill the spawner
			err := pproc.Kill()
			if err != nil {
				log.Println("Error finding :", err)
				return
			}

			fmt.Println("flare cli (spawner) killed")
			time.Sleep(1 * time.Second)
		}
	}

	// TODO: implement file checking
	if err := updates.EnsureUpdateFilesExist(); err != nil {
		log.Println("Error in ensuring core release files exist: ", err)
		os.Exit(1)
	}

	// update the system by copying and replacing
	fmt.Println("updating system..")
	if err := updates.Update(); err != nil {
		log.Println("Error updating system:", err)
		os.Exit(1)
	}

	// run the copied flare
	fmt.Println("running new flare..")
	if err := updates.ExecuteFlare(); err != nil {
		log.Println("Error executing new flare cli:", err)
		os.Exit(1)
	}

	// finish update
	fmt.Println("Core System Updated Successfully!")
	os.Exit(0)
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

    fix-workspace                       Re-generate the go.work file

    install-go  <install path>          Install Go to the given path. If install path argument is not defined, then it will install in
                                        the "$GO_CUSTOM_PATH" if defined, else it will install in "go" directory under the
                                        current working directory.

    update                              Updates the flare system
`
}
