# The `plugin.json` File

## Fields

The `plugin.json` file contains the metadata of the plugin. It is a JSON file located in the root directory of a plugin. It contains the following fields:

### name
The name of the plugin.

### package
The package name of the plugin. It should be unique and in reverse domain format. Example: `com.mydomain.myplugin`.

### description
The description of the plugin.

### version

The version of the plugin. It should follow the [Semantic Versioning](https://semver.org/) format. Example: `1.0.0`

### system_packages

List of system packages required by the plugin. These are installed via `opkg`
(the install runs `opkg update` then `opkg install` for any not already present).

Because `opkg` needs the package feed, system packages are installed **when the
machine has internet** — not blindly during the offline part of boot:

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

Optional path (relative to the plugin root) to a shell script run **after**
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

Optional path (relative to the plugin root) to a shell script run **after** the
plugin's `Init`. Also **gated on internet connectivity**. Use it for work that
should happen once the plugin is already running; anything `Init` itself depends on
belongs in `preinstall` instead.

```sh
#!/bin/sh
# Only run the real setup on a production machine; no-op everywhere else.
[ "${GO_ENV:-production}" = "production" ] || exit 0
pip3 install --no-cache-dir SomeLibrary
```

Both `preinstall` and `postinstall` run through the shell with the plugin's
source directory as the working directory, and inherit the server's
stdout/stderr. A non-zero exit fails the install.

#### The `GO_ENV` environment variable

Install scripts are run with a `GO_ENV` environment variable set to the current
build environment — one of `development`, `sandbox`, `staging`, or
`production`. Scripts **should guard on it** so they remain a clean no-op in
development (where machine tooling such as `opkg`/`pip3` is absent and running
machine setup would otherwise fail the boot-time install):

```sh
[ "${GO_ENV:-production}" = "production" ] || exit 0
```

### sdk

The minimum sdk version that the plugin supports. The supported SDK versions are available in [flarehotspot/devkit](https://github.com/flarewifi/devkit/releases) repository.

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
