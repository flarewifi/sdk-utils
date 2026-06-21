
# Creating a Plugin

## Using `create-plugin` command {#create-plugin}

To create a new plugin, open a terminal and navigate inside the devkit directory.

If you are using Windows `CMD` or `PowerShell`, type:
```cmd title="PowerShell"
.\scripts\flare.bat create-plugin
```

If you are using Linux/Mac/WSL, type:
```sh title="Terminal"
./scripts/flare.sh create-plugin
```

Follow the instructions in the command prompt and enter any necessary details for your plugin. Below are the required details for your plugin:

### Package Name

This is the primary identifier for your plugin. It must be unique and follow reverse domain naming conventions, such as `com.mydomain.myplugin`. The package name should be entirely lowercase and can only include periods, underscores, or hyphens (`.` `_` `-`).

### Plugin Name

This is the official name of your plugin, for example: "System Monitor".

### Description

Please provide a concise description of your plugin. This should briefly explain its purpose.

---

## Cloning an existing plugin {#cloning-plugin}

If you need to develop an existing plugin, open a terminal and navigate to `data/plugins/local` folder inside the devkit directory. Then clone the plugin:

```sh title="Terminal"
cd [devkit-root]/data/plugins/local
# Replace the URL with the URL of the plugin you want to clone
git clone https://github.com/flarewifi/com.flarego.sample
```

Now you can start developing your plugin.

---

## Plugin file structure {#file-structure}

After scaffolding, your plugin lives at `data/plugins/local/[your-plugin-package]` (e.g. `data/plugins/local/com.mydomain.myplugin`). Inside, you will find these required Go files at the plugin root:

| File | Owner | Build tag | Purpose |
|---|---|---|---|
| `main.go` | **You** | `//go:build !mono` | The `.so` build entry point. Defines `func main(){}` and `Init(api)`. This is where most plugin code goes. |
| `main_mono.go` | You (auto-generated if missing) | `//go:build mono` | Customization slot for mono builds. Auto-generated from `main.go` on first scaffold; after that it is yours to edit. |
| `system/main.go` | **Generator (do NOT edit)** | `//go:build !mono` | Mirror of `main.go` with package renamed to `system`. Used when the plugin is statically linked into the core binary for non-mono system plugin builds. |
| `system/main_mono.go` | **Generator (do NOT edit)** | `//go:build mono` | Mirror of `main_mono.go` with package renamed to `system`. Used when the plugin is statically linked into a mono build of the core binary. |

You will also see the standard plugin scaffold:

```
[your-plugin-package]/
  main.go              # author — !mono entry
  main_mono.go         # author — mono entry
  system/
    main.go            # generated — mirror of main.go
    main_mono.go       # generated — mirror of main_mono.go
  plugin.json          # plugin metadata
  go.mod
  resources/
    migrations/        # SQL migrations
    queries/           # sqlc query definitions
    views/             # templ templates
    assets/            # static assets
    translations/      # i18n
  LICENSE.txt
  .gitignore
```

!!! warning "Do not edit `system/` files"
    `system/main.go` and `system/main_mono.go` are derived copies of your root `main.go` and `main_mono.go`. The generator rewrites them on every prep run to match the root. Any edits you make to `system/` files are silently overwritten. Edit `main.go` (or `main_mono.go`) instead — the changes propagate to `system/` automatically.

!!! info "Why does my plugin need four files?"
    Plugins serve two purposes: they can be loaded dynamically as `plugin.so` files (which requires `package main` at the root) AND statically linked into the core binary for "mono" builds and system plugins (which requires an importable, non-`main` package). The two `system/` files solve the static-link case by exposing the same code under `package system`. The two root files cover both the dynamic-load case and let you customize behavior per build mode.

---

## The `main.go` file

This file contains the `Init` function which is called when your plugin gets loaded. Below is the initial content of `main.go`:

```go title="main.go"
//go:build !mono

package main

import (
    sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
    // Your plugin code here
    return nil
}
```

!!! note
    The `api` variable is an instance of the [IPluginApi](../api/plugin-api.md), the root API of the Flarewifi SDK. Throughout the documentation, when you see the variable `api`, it refers to [IPluginApi](../api/plugin-api.md).

!!! warning "Keep the `//go:build !mono` tag"
    The `//go:build !mono` tag on the first line is mandatory. It tells Go to compile this file only for non-mono builds (`.so` builds for local/devel/git/store plugins). Removing it will cause a build-tag conflict with `main_mono.go`, which has `//go:build mono`. The generator will re-add the tag if you accidentally remove it, but it is best to leave it in place.

---

## Per-mode customization {#per-mode}

For most plugins, `main.go` and `main_mono.go` have identical bodies — both just call into the same `Init` logic. You only need to make them differ when your plugin behaves differently between non-mono and mono builds (e.g. some features are disabled in mono mode).

To customize, edit `main_mono.go` to provide a different `Init` body:

```go title="main_mono.go"
//go:build mono

package main

import (
    sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
    // mono-specific initialization (e.g. skip features that
    // don't apply when bundled into the core binary)
    return nil
}
```

After editing `main_mono.go`, the next prep run will regenerate `system/main_mono.go` to mirror it — so the mono static-link path automatically picks up your changes.

---

## Sharing code across mono and non-mono {#common-subpackage}

If both `main.go` and `main_mono.go` need to call shared helpers (or if you want to keep `Init` thin and put the real logic in helpers), put the shared code in a subpackage. By convention this is called `common/`:

```
[your-plugin-package]/
  common/
    init.go            # //go:build !mono   package common   InitPlugin()
    init_mono.go       # //go:build mono    package common   InitPlugin()
  main.go              # delegates to common.InitPlugin
  main_mono.go         # delegates to common.InitPlugin
  system/main.go       # generated — also delegates to common.InitPlugin
  system/main_mono.go  # generated — also delegates to common.InitPlugin
```

Example contents:

```go title="common/init.go"
//go:build !mono

package common

import (
    "com.mydomain.myplugin/app/routes"
    sdkapi "sdk/api"
)

func InitPlugin(api sdkapi.IPluginApi) error {
    routes.AdminRoutes(api)
    return nil
}
```

```go title="common/init_mono.go"
//go:build mono

package common

import (
    sdkapi "sdk/api"
)

func InitPlugin(api sdkapi.IPluginApi) error {
    // mono variant (often a no-op)
    return nil
}
```

```go title="main.go"
//go:build !mono

package main

import (
    "com.mydomain.myplugin/common"
    sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
    return common.InitPlugin(api)
}
```

```go title="main_mono.go"
//go:build mono

package main

import (
    "com.mydomain.myplugin/common"
    sdkapi "sdk/api"
)

func main() {}

func Init(api sdkapi.IPluginApi) error {
    return common.InitPlugin(api)
}
```

The root files become thin delegates. The `system/` mirror is regenerated from them and inherits the same delegation. The `common/` subpackage is the single source of truth for the per-mode initialization logic. Go's build-tag filtering automatically selects `common/init.go` under `!mono` and `common/init_mono.go` under `mono`.

You can use any name you like for the subpackage (e.g. `pluginimpl`, `lib`, `helpers`) — `common` is just convention. The directory becomes a regular Go subpackage of your plugin module; no `go.mod` edits are required.

---

## What about the `app/`, `resources/`, `db/` directories? {#other-dirs}

These are not Go-source directories at the root level — they hold the plugin's controllers, templates, assets, and generated code. They are unaffected by the four-file contract:

- **`app/`** — Plugin controllers, services, routes, views (HTML/JS/CSS). Author-owned.
- **`resources/`** — Migrations (`.sql`), sqlc queries (`.sql`), templ templates (`.templ`), static assets, translations (`.json`).
- **`db/queries/`** — Generated sqlc Go code from `resources/queries/*.sql`. Do not edit; regenerated by sqlc.

---

## Troubleshooting

For Linux users, you must change the file permissions to fix errors in your code editor:
```sh title="Terminal"
sudo chown -R $USER .
```

For MacOS users, if you encounter `Too many open files in system` error, you can fix this by cleaning the Go build cache and fixing the file permissions:

```sh title="Terminal"
go clean -cache
sudo chown -R $USER .
```

### `plugin contract violation: <file> is missing`

Your plugin is missing one of the four required files (`main.go`, `main_mono.go`, `system/main.go`, or `system/main_mono.go`). The most common cause is accidentally deleting a generated `system/` file. Fix it by running prep again (`./start-dev.sh` or restarting the dev container) — the generator will recreate it from the root files.

### `undefined: initPlugin` (or similar) in `system/main.go`

The mirror generator copies your root `main.go` verbatim (with package renamed to `system`). If your root `main.go` references helpers defined in sibling files at the root (e.g. `helpers.go`), those helpers are NOT mirrored into `system/`. Move the shared helpers into a `common/` subpackage (see [Sharing code across mono and non-mono](#common-subpackage)) and have both root files call into `common.YourHelper()`.

### My edit to `system/main.go` keeps getting overwritten

That is expected. `system/main.go` is a derived mirror — edit `main.go` at the plugin root instead, and the generator will regenerate `system/main.go` to match.

---

## Related

- [IPluginApi](../api/plugin-api.md) — The root API passed to `Init`; entry point for all SDK functionality
- [plugin.json](../api/plugin.json.md) — Plugin metadata: name, package, version, permissions, and install scripts
- [Plugin Info](../api/plugin-info.md) — `api.Info()` for reading the plugin's own metadata at runtime
- [Routes and Navigation](./routes-and-navigation.md) — Registering HTTP routes inside `Init`
