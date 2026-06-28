# Flarewifi Developer Docs

Flarewifi is a **plugin-based WiFi hotspot platform** for OpenWRT machines. The core stays deliberately small — nearly every feature, from captive portals to payment gateways, ships as a **plugin** you can build, install, and sell.

These docs cover everything you need to extend a Flarewifi machine: the plugin SDK, step-by-step guides, and the full API reference.

[Get Started](./guides/getting-started.md){ .md-button .md-button--primary }
[API Reference](./api/plugin-api.md){ .md-button }

## What you can build

- **Captive portals & themes** — control the look of both the admin dashboard and the user-facing portal with [admin and portal themes](./api/themes-api.md).
- **Access & voucher flows** — sell time- or data-based access, [generate vouchers](./api/voucher-api.md), and [accept payments](./tutorials/accepting-payments.md).
- **Network & bandwidth control** — manage per-client bandwidth, [sessions](./api/sessions-mgr-api.md), and [firewall rules](./api/firewall-api.md).
- **Background services & events** — react to session, client-device, and internet-connectivity [events](./api/events-api.md).

## Why build on Flarewifi?

- **Go SDK** with type-safe database access (`sqlc`) and server-rendered views (`templ`).
- **Minimal core, plugin everything** — your plugin owns its own tables, routes, assets, and translations without ever modifying the core.
- **Get paid** — publish plugins and earn developer commissions on every purchase.

## Find your way around

- **[Tutorials](./guides/getting-started.md)** — guided walkthroughs from [creating your first plugin](./guides/creating-a-plugin.md) to [handling events](./guides/handling-events.md).
- **[Examples](./tutorials/accepting-payments.md)** — complete, real-world plugins such as [accepting payments](./tutorials/accepting-payments.md) and [building a theme](./tutorials/creating-a-theme/index.md).
- **[API Reference](./api/plugin-api.md)** — every SDK interface, from [`IPluginApi`](./api/plugin-api.md) to [`IThemesApi`](./api/themes-api.md).
