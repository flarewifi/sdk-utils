# Plugin Scripts

Plugins can ship small shell scripts that run around installation (and removal)
to prepare or clean up the machine — installing system packages, configuring the
firewall, and so on. This guide covers the concepts: **where scripts must
live**, **when they run**, **how to write them safely**, and **how a running
plugin can invoke one at runtime**.

For the `plugin.json` field reference and exact lifecycle ordering, see
[`plugin.json`](../api/plugin.json.md).

## The install hooks

A plugin declares its scripts and system packages in `plugin.json`
(`system_packages`, `preinstall`, `postinstall`):

- **`system_packages`** — OS packages installed via the machine's package
  manager (`opkg`/`apk`) before your scripts run.
- **`preinstall`** — runs after `system_packages`, **before** your plugin's
  `Init`. Put anything `Init` depends on here.
- **`postinstall`** — runs after `Init` (skipped if `Init` fails).

All three are gated on internet connectivity and run **once per plugin version**.

## The uninstall hooks

A plugin can also declare `preuninstall`/`postuninstall`, run when it's
removed — the mirror image of the install hooks, for undoing whatever a
plugin changed **outside its own install directory** (that directory is
deleted as the last step of removal regardless, so nothing inside it needs
manual cleanup). Unlike the install hooks, these are **not** gated on
internet connectivity and **not** version-tracked — see the
[`plugin.json` reference](../api/plugin.json.md#preuninstall) for the exact
differences.

## Where scripts must live

!!! danger "Only shipped directories reach the install dir"
    A plugin's Go source under `app/` is **not** copied into the install
    directory — it is compiled into `plugin.so`. Keep scripts, and any file they
    read, under a **shipped** path (`scripts/`, or a `resources/` subtree). A
    file that isn't shipped works on the first install (run from your source
    tree) but is missing on a later boot re-run (run from the install dir).

Put install scripts under **`scripts/`**.

## When scripts run

Scripts run in two situations, both requiring the machine to be online (the
package manager needs its feed):

- **Runtime install** (from the dashboard) — inline; working directory is your
  **source** tree.
- **Baked-in plugins** (an os_image's first boot) — driven by the online monitor
  when internet arrives (see
  [`OnInternetEvent`](../api/events-api.md#oninternetevent)), retried on the next
  reconnect if it failed; working directory is the **install** dir.

Each phase records the plugin **version** it completed for, so it re-runs only on
a version bump (an upgrade).

## Writing a safe script

- **Guard on `GO_ENV`** — scripts inherit `GO_ENV` (`development`, `sandbox`,
  `staging`, `production`). The dev container has no `opkg`/`uci`, so make the
  script a no-op unless the value is `production`/`staging`.
- **Be idempotent** — a failed pass is retried and an upgrade re-runs the script,
  so it must tolerate running again (e.g. prefer named, set-style `uci` sections
  over appends).
- **Don't assume the working directory** — it differs between install and boot
  re-run, so reference sibling files relative to the script (`$(dirname "$0")`),
  not by cwd.
- **Mind exit codes** — a non-zero exit fails the install; the whole pass is
  retried on the next reconnect. Swallow best-effort failures explicitly.

## Running a shipped script at runtime

Because `scripts/` is shipped, a running plugin can invoke one of its scripts
later — for example to re-apply configuration on a user action. Resolve the path
from the plugin's install root with [`api.Dir()`](../api/plugin-api.md)
(`filepath.Join(api.Dir(), "scripts", "...")`) and run it through the shell.

This lets one idempotent script be the single source of truth for both the
install-time hook and a runtime code path, so the two can never drift. Split
machine-only logic behind a `dev` build tag so the dev container stubs it out.
