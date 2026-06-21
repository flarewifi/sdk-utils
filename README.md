# Flare Hotspot

Flare Hotpost core repository.

# System Requirements
- Make
- Docker
- Go

# Installation

Clone the project and prepare the development environment.
```sh
git clone git@github.com:flarewifi/flarewifi.git
cd flarewifi
git checkout development
cp go.work.default go.work
```

Unzip the `openwrt-files.zip` file.

```sh
rm -rf ./data/openwrt-files
mkdir data
unzip openwrt-files.zip -d data/openwrt-files
```

Create `hosts.json` for mocking device MAC or IP:

```sh
cp hosts.json.sample hosts.json
```

# Start the server

```sh
make
```
## HTTPS (captive portal + admin)

**Both the captive portal and the admin dashboard always run over HTTPS**, in dev
and in production. Every plain-HTTP request is redirected to HTTPS (the HTTP
listener stays up only to redirect, which is also how OS captive-portal probes are
caught — see `middlewares.ForceHTTPS`). The redirect **target host depends on the
page**:

| You open (HTTP) | Redirects to (HTTPS) | Why |
|---|---|---|
| `http://localhost:3000/` (portal) | `https://captive.flare-local.com:3443/` | portal domain → **valid cert** |
| `http://localhost:3000/admin` (admin) | `https://localhost:3443/admin` | **same host** → admin works by IP, cert-name warning expected |

- **Portal/captive** traffic goes to the portal domain: **`captive.flare-local.com`**
  in dev (fixed), the configured **`custom_domain`** in production (e.g.
  `captive.flarewifi.com`), so it is served with the valid cloud-issued cert.
- **Admin** stays on the **same host**, so the dashboard is reachable by raw IP with
  **no domain required** (e.g. `https://10.0.0.1/admin`); the served cert is for the
  portal domain, so a name-mismatch warning is expected — accept it.

The dev HTTPS port is **3443** (production uses 443).

> **Dev:** add the portal hostname to `/etc/hosts` so the browser resolves it to
> the local container:
> ```
> 127.0.0.1 captive.flare-local.com
> ```

> **Production — DNS requirement:** the portal domain (`captive.flarewifi.com`, or
> whatever `custom_domain` is set to) **must resolve to the device's LAN gateway IP
> `10.0.0.1`**. The device's own dnsmasq does this with a split-horizon entry, so
> any client on the device's network — and the admin operator — reach the local
> server (with its valid cert) instead of the public internet. If the domain does
> not point to `10.0.0.1`, the HTTPS redirect lands off-device and the portal/admin
> become unreachable.

### Portal certificate & the Cloudflare token

The portal certificate is **not** generated on the device. The cloud
([flare-server](https://github.com/flarewifi/flare-server)) issues a real
Let's Encrypt certificate for the portal hostname (`captive.flare-local.com` in
dev) using the **ACME DNS-01 challenge against the Cloudflare zone**, and the
machine fetches and installs it via cloud-sync. The device only falls back to a
self-signed cert when no cloud-issued cert is available yet.

For dev HTTPS to serve a proper (Let's Encrypt **staging**) cert, the
**flare-server** must have a valid Cloudflare API token set — a token with
*DNS:Edit* permission on the `flare-local.com` zone:

```sh
# in the flare-server repo's .env
CLOUDFLARE_TOKEN=<cloudflare-api-token-with-DNS-edit-on-flare-local.com>
# optional: switch from Let's Encrypt staging (default) to production
# ACME_DIRECTORY_URL=https://acme-v02.api.letsencrypt.org/directory
```

Without it, `flare-server` logs `portalcert: CLOUDFLARE_TOKEN is not set` and no
cert is issued, so the device serves the self-signed fallback (browser shows an
"untrusted certificate" warning). With the **staging** directory (the dev
default) the chain is real but signed by Let's Encrypt's staging root, so the
browser still warns until the cloud is pointed at the production ACME directory.

The database can be managed at [http://localhost:3001](http://localhost:3001)

# Device Activation (Development Environment)

These steps activate a machine locally when running flarehotspot alongside [flare-server](https://github.com/flarewifi/flare-server). This is for **dev setup only** — not production.

### Prerequisites

1. **Hostfile configured** — add all required entries to `/etc/hosts`. See the [flare-server README](https://github.com/flarewifi/flare-server#installation) for the full list.
2. **flare-server running** — `make servers` in the flare-server directory.
3. **Docker running** — both projects rely on Docker Compose.

### Steps

1. **Set up user permissions** — log into [superuser.flare-local.com](http://superuser.flare-local.com), ensure your user has the required permission flags (at minimum `is_internal`).
2. **Create a B2B Partner** — on the superuser dashboard, create a new B2B Partner and note the **Brand ID**.
3. **Configure `os_release.json`** — in `openwrt-files/etc/`, copy the correct variant for your architecture (`os_release.x86.json` or `os_release.arm64.json`) to `os_release.json`, then replace the `brand_id` value with the Brand ID from step 2.
4. **Start both servers** — run `make servers` in the flare-server terminal and `make` in the flarewifi terminal.
5. **Activate the machine** — go to [internal.flare-local.com](http://internal.flare-local.com), find the machine by its `machine_id`, and activate it (fill in user details, tick "Is activated", save).

> **Tip:** When extracting `openwrt-files.zip`, watch out for nested `openwrt-files/openwrt-files/` directories. The directory containing `etc/` must be at the project root.

# Documentation

To view the documentation locally, visit [http://localhost:3002](http://localhost:3002).

To build the documentation to be uploaded to the docs website:

```sh
make docs-build
```

Then you can find the built documentation in the `sdk/mkdocs/site` directory.


# UI Testing with Playwright

The project includes Playwright MCP for automated UI testing and verification. Playwright is installed in `.opencode/` and accessible to all developers.

## Installation

Playwright is already configured in `.opencode/package.json`. To install or update dependencies:

```sh
cd .opencode
npm install
npx playwright install chromium
```

## Usage

Playwright can be used to test the application running at `http://localhost:3000`:

- Navigate to pages and take snapshots
- Click buttons and fill forms
- Verify UI flows work correctly
- Test both admin (Bootstrap 5) and portal (Bootstrap 3) interfaces
- Verify translations display correctly

All Playwright outputs (screenshots, reports) should be saved to `.tmp/playwright/`.

## Example Workflow

1. Start the application: `make` (from project root)
2. Use Playwright MCP tools to:
   - Navigate to a page (`browser_navigate`)
   - Inspect the page structure (`browser_snapshot`)
   - Interact with elements (`browser_click`, `browser_type`)
   - Take screenshots for verification (`browser_take_screenshot`)
3. Close the browser when done (`browser_close`)

## Notes

- Portal/login pages use Bootstrap 3.4.1
- Admin/dashboard pages use Bootstrap 5.3.3
- Remember to close the browser after testing to free resources

# Plugins

To clone an existing plugin, run the following command:

```sh
cd ./plugins/local
git clone git@github.com:flarewifi/com.flarego.example.git
```

To create a new plugin, run the following command:

```sh
cd flarewifi
./scripts/flare.sh create-plugin
```

# Running in OpenWrt

This guide is based on the official [OpenWrt Image Builder Guide](https://openwrt.org/docs/guide-user/additional-software/imagebuilder).

Download the image builder for [x86](https://archive.openwrt.org/releases/23.05.4/targets/x86/64/). Then extract the contents of the image builder to your desired location.

After extracting the image builder, navigate to the directory where you extracted it and run the following command:
```sh
export PACKAGES="golang   sudo   usbutils   ntpd   tc   nftables   kmod-ifb   kmod-crypto-ecb   kmod-crypto-xts   kmod-crypto-seqiv   kmod-crypto-misc   kmod-crypto-user   cryptsetup   kmod-fs-ext4   e2fsprogs   losetup   pgsql-server   gcc   tar   ar   git-http   ca-bundle   kmod-usb-core   kmod-usb2   kmod-usb3   kmod-usb-net-asix   kmod-usb-net-asix-ax88179   kmod-usb-net-rtl8152   kmod-usb-net-smsc95xx   kmod-usb-net-cdc-ether   kmod-usb-net-dm9601-ether   kmod-usb-net-kaweth   kmod-usb-net-pegasus   kmod-usb-net   kmod-usb-net-rndis   parted   losetup   resize2fs   block-mount   zram-swap"

make image \
    PROFILE=generic \
    PACKAGES=${PACKAGES} \
    ROOTFS_PARTSIZE=5120
```

Once the image is built, you can find it in the `bin/targets/x86/64/` directory. The file will be named something like `openwrt-23.05.4-x86-64-generic-squashfs-combined.img.gz`. Extract the image to a `.img` file.

To run the generated image in VirtualBox, you have to convert the image to a VDI file. You can do this using the `qemu-img` tool:
```sh
qemu-img convert -f raw -O vdi openwrt-23.05.4-x86-64-generic-squashfs-combined.img openwrt.vdi
```
Then, you can create a new VirtualBox VM and attach the `openwrt.vdi` file as the hard disk.

Once the VM is running, execute the following command inside the VM to fix compilation issues:

```sh
ar -rc /usr/lib/libpthread.a
ar -rc /usr/lib/libresolv.a
ar -rc /usr/lib/libdl.a
```
