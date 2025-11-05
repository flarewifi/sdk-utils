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


# Start the server

```sh
make
```
Now you can browse the portal at [http://localhost:3000](http://localhost:3000)

The admin dashboard can be accessed at [http://localhost:3000/admin](http://localhost:3000/admin)

The database can be managed at [http://localhost:3001](http://localhost:3001)

# Documentation

To view the documentation locally, visit [http://localhost:3002](http://localhost:3002).

To build the documentation to be uploaded to the docs website:

```sh
make docs-build
```

Then you can find the built documentation in the `sdk/mkdocs/site` directory.


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
