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
rm -rf ./openwrt-files
unzip openwrt-files.zip -d openwrt-files
```


# Start the server

```sh
make
```
Now you can browse the portal at [http://localhost:3000](http://localhost:3000)

The admin dashboard can be accessed at [http://localhost:3000/admin](http://localhost:3000/admin)

The database can be managed at [http://localhost:8081](http://localhost:3001)

# Documentation

To view the documentation locally, visit [http://localhost:8000](http://localhost:3002).

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
