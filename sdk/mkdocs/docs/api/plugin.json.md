# The `plugin.json` File

## Fields

The `plugin.json` file contains the metadata of the plugin. It is a JSON file located in the root directory of a plugin. Only the fields below are recognized — any other keys are silently ignored.

### name
*(required)* The name of the plugin.

### package
*(required)* The package name of the plugin. It should be unique and in reverse domain format. Example: `com.mydomain.myplugin`.

### description
*(required)* The description of the plugin.

### version

*(required)* The version of the plugin. It should follow the [Semantic Versioning](https://semver.org/) format. Example: `1.0.0`

The version is also what makes the network-dependent install phases
(`system_packages`, `preinstall`, `postinstall`) run **once per version**: each
phase records the version it last completed for, so bumping the version re-runs
them on the next reconnect (and leaving it unchanged does not).

### system_packages
*(optional)*

List of system packages required by the plugin. These are installed via the
machine's package manager — `opkg`, or `apk` on newer OpenWRT — detected
automatically (the install updates the package index, then installs any package
not already present).

Because the package manager needs its feed, system packages are installed **when
the machine has internet** — not blindly during the offline part of boot:

- **Runtime install** (from the dashboard): installed inline, since the install
  itself proves the machine is online.
- **Baked-in plugins** (os_image first boot): installed by the core's **online
  monitor** the moment internet becomes available (see
  [`OnInternetEvent`](./events-api.md#oninternetevent)), and retried on the next
  reconnect if a previous attempt failed.

Installation is recorded per plugin version, so the `opkg` work runs once per
version and is not repeated on every boot.

### Lifecycle ordering

A plugin goes through these phases, in order:

1. **Load** — the plugin's compiled code is mapped into the process and its `Init`
   entry point is resolved. This happens at boot, **offline** — it needs no network.
2. **`system_packages`** — installed via `opkg` (needs internet).
3. **`preinstall`** — runs after `system_packages` (needs internet).
4. **`Init`** — the plugin's `func Init(api)` runs, **after `preinstall`**, so it
   can rely on its system packages and preinstall setup being in place.
5. **`postinstall`** — runs after `Init`.

On a machine flashed offline, phases 2–5 are held until the **online monitor** sees
internet (see [`OnInternetEvent`](./events-api.md#oninternetevent)). A plugin that
declares no `system_packages` and no `preinstall` has nothing to wait for, so its
`Init` runs at boot — that's how the captive portal comes up without a network.
Each network-dependent phase is recorded per plugin version, so it runs once per
version and is retried on the next reconnect if it failed.

### preinstall

*(optional)* Path (relative to the plugin root) to a shell script run **after**
`system_packages` are installed and **before the plugin's `Init`**. Because `Init`
waits for it, `preinstall` is the right place for anything `Init` depends on — for
example `pip`-installing a Python library (not in the `opkg` feed) that the plugin
imports at startup:

```sh
#!/bin/sh
# Only run the real setup on a production machine; no-op everywhere else.
[ "${GO_ENV:-production}" = "production" ] || exit 0
pip3 install --no-cache-dir SomeLibrary
```

Like `system_packages`, `preinstall` is **gated on internet connectivity**, so a
script that fetches something over the network can rely on connectivity being
present when it runs.

### postinstall

*(optional)* Path (relative to the plugin root) to a shell script run **after** the
plugin's `Init`. Also **gated on internet connectivity**. Use it for work that
should happen once the plugin is already running; anything `Init` itself depends on
belongs in `preinstall` instead.

Because it runs after `Init`, `postinstall` is **skipped when `Init` fails** — the
whole provisioning pass (system packages → preinstall → Init → postinstall) is
retried on the next reconnect instead.

```sh
#!/bin/sh
# Only run the real setup on a production machine; no-op everywhere else.
[ "${GO_ENV:-production}" = "production" ] || exit 0
pip3 install --no-cache-dir SomeLibrary
```

Both `preinstall` and `postinstall` run through the shell and inherit the
server's stdout/stderr. A non-zero exit fails the install.

The working directory is the plugin's **source** tree on a dashboard install but
the plugin's **install** directory on a baked-in plugin's boot-time run — so
reference sibling files relative to the script itself
(`"$(dirname "$0")/helper.sh"`), and keep any helper the script reads under a
**shipped** path such as `scripts/` (a plugin's `app/` Go source is not copied
into the install directory). See the
[Plugin Scripts guide](../guides/plugin-scripts.md) for the full set of
gotchas — shipping, idempotency (scripts may re-run on reconnect/upgrade), and
invoking a shipped script from Go at runtime.

#### The `GO_ENV` environment variable

Install scripts are run with a `GO_ENV` environment variable set to the current
build environment — one of `development`, `sandbox`, `staging`, or
`production`. Scripts **should guard on it** so they remain a clean no-op in
development (where machine tooling such as `opkg`/`pip3` is absent and running
machine setup would otherwise fail the boot-time install):

```sh
[ "${GO_ENV:-production}" = "production" ] || exit 0
```

### preuninstall

*(optional)* Path (relative to the plugin root) to a shell script run when the
plugin is removed, **before** its DB down-migrations run and its metadata
record is deleted — i.e. while the plugin is still fully intact.

### postuninstall

*(optional)* Path (relative to the plugin root) to a shell script run when the
plugin is removed, **after** its DB down-migrations and metadata removal but
**immediately before its install directory is deleted** — the last moment the
script file itself still exists on disk to be run.

Use `preinstall`/`postinstall` to prepare things `Init` depends on; use
`preuninstall`/`postuninstall` for the mirror image — undoing whatever a
plugin changed **outside its own install directory** (a system service file
under `/etc/init.d`, a firewall include, a crontab entry, files moved to fixed
system paths by `postinstall`). Nothing needs to clean up files *inside* the
install directory — removing it is the last step of uninstall regardless.

Uninstall scripts differ from `preinstall`/`postinstall` in three ways:

- They are **not gated on internet connectivity** — uninstall is a purely
  local operation (delete DB rows/files, run a script), so it runs at the next
  boot regardless of network state.
- They are **not "once per version"** — there is no version-pinned marker the
  way install phases have one (tracking "did this already run" for a plugin
  that no longer exists has no boot to check it against). A script must
  tolerate being invoked and, on failure, potentially retried on a later boot
  (the plugin stays marked for removal until `UninstallPlugin` fully
  succeeds).
- Uninstalling is **always** deferred to the next boot — clicking "uninstall"
  in the dashboard only marks the plugin for removal; unlike `preinstall`/
  `postinstall`, there is no dashboard-triggered inline run, so the working
  directory is always the plugin's **install** directory, never a source tree.

```sh
#!/bin/sh
[ "${GO_ENV:-production}" = "production" ] || exit 0
# Undo something this plugin set up outside its own install directory.
rm -f /etc/some-external-config.conf
```

### sdk

*(optional)* The minimum sdk version that the plugin supports. The supported SDK versions are available in [flarehotspot/devkit](https://github.com/flarewifi/devkit/releases) repository.

## Example

Below is an example of a `plugin.json` file:

```json
{
    "name": "My Plugin",
    "package": "com.mydomain.myplugin",
    "description": "This is my plugin description",
    "version": "0.0.1",
    "system_packages": [],
    "preinstall": "scripts/preinstall.sh",
    "postinstall": "scripts/postinstall.sh",
    "sdk": "1.0.0"
}
```
