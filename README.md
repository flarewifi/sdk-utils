# Flare Hotspot

Flare Hotpost core repository.

# System Requirements
- Make
- Docker
- Go

# Installation

Clone the project and prepare the development environment.
```sh
git clone git@github.com:flarehotspot/flarehotspot.git
cd flarehotspot
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
Now you can browse the portal at [http://localhost:3000](http://localhost:3000)

The admin dashboard can be accessed at [http://localhost:3000/admin](http://localhost:3000/admin)

The database can be managed at [http://localhost:3001](http://localhost:3001)

# Device Activation (Development Environment)

These steps activate a machine locally when running flarehotspot alongside [flare-server](https://github.com/flarehotspot/flare-server). This is for **dev setup only** — not production.

### Prerequisites

1. **Hostfile configured** — add all required entries to `/etc/hosts`. See the [flare-server README](https://github.com/flarehotspot/flare-server#installation) for the full list.
2. **flare-server running** — `make servers` in the flare-server directory.
3. **Docker running** — both projects rely on Docker Compose.

### Steps

1. **Set up user permissions** — log into [superuser.flare-local.com](http://superuser.flare-local.com), ensure your user has the required permission flags (at minimum `is_internal`).
2. **Create a B2B Partner** — on the superuser dashboard, create a new B2B Partner and note the **Brand ID**.
3. **Configure `os_release.json`** — in `openwrt-files/etc/`, copy the correct variant for your architecture (`os_release.x86.json` or `os_release.arm64.json`) to `os_release.json`, then replace the `brand_id` value with the Brand ID from step 2.
4. **Start both servers** — run `make servers` in the flare-server terminal and `make` in the flarehotspot terminal.
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
git clone git@github.com:flarehotspot/com.flarego.example.git
```

To create a new plugin, run the following command:

```sh
cd flarehotspot
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
