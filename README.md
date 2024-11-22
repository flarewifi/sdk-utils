
# Flare Hotspot
Flare Hotpost core repository.

# System Requirements
- Docker

# Installation

Clone the project and prepare the development environment.
```sh
git clone git@github.com:flarehotspot/flarehotspot.git
cd flarehotspot
git checkout development
cp go.work.default go.work
git submodule update --init --recursive
```

Pull the latest changes for all the submodules.
```sh
for p in $(ls plugins); do echo plugins/$p; for subp in $(ls plugins/${p}); do echo plugins/$p/$subp; cd plugins/$p/$subp; git checkout development && git pull; cd ../../..; done; done
```

Install node modules.

```sh
npm install
```

Unzip the `openwrt-files.zip` file.

```sh
rm -rf ./openwrt-files
unzip openwrt-files.zip -d openwrt-files
```

# Installing Go
```sh
curl -sSL https://github.com/moovweb/gvm/raw/master/binscripts/gvm-installer | bash
gvm install "go$(cat .go-version)"
cd ../flarehotspot # load go version
```

# Start the server

```sh
docker compose up --build
```
Now you can browse the portal at [http://localhost:3000](http://localhost:3000)

The admin dashboard can be accessed at [http://localhost:3000/admin](http://localhost:3000/admin)

The default admin access is:
```
username: admin
password: admin
```

# Flare CLI

Install the `flare` CLI:
```sh
go run ./core/cmd/build-cli/main.go
./bin/flare -h
```

# Documentation

Make sure `pipx` is available in your system and install the following packages:

```sh
pipx install mkdocs-material --include-deps
```

Then you can serve the local documentation server:

```sh
cd flarehotspot
make docs-serve
```

To build the documentation to be uploaded to the docs website:

```sh
make docs-build
```

---

# Steps in implementing git subtree for `go-utils`

## Split the utils to a `git subtree`.

```sh
git subtree split --prefix sdk/utils -b go-utils
```

This will create a new branch called `go-utils` which can be pushed to a git repo.

## Add the necessary `go.mod` file for making the `go-utils` a standalone library.

Example:
```go
module github.com/flarehotspot/go-utils

go 1.22.0
```

## Add the remote url of `flarehotspot/go-utils`

```sh
git remote add go-utils git@github.com:flarehotspot/go-utils.git
```

## Push the `go-utils` branch to a remote git repo.
```sh
git push go-utils go-utils:main
```

# Pushing changes to `go-utils`

```sh
# command guide
# git subtree push --prefix <utils dir name> <go-utils remote name or url> <desired local branch to push>
# don't worry, this will only push the changes inside the `utils` and not the entire local branch

# actual command
git subtree push --prefix sdk/utils go-utils development # or your desired local branch e.g. feat/utils-subtree
```

# Persist changes

For the changes to persist in other codebases that uses the go library, head over to the github or even to the local cloned repo of `go-utils` and create a git tag.

```sh
git checkout go-utils
git tag vx.x.x # creates a tag to the latest commit of the current branch
git push --tags # pushes the created tag
```

Then, update the `go-utils` library by specifying the version of the newly pushed tag.
```sh
go get -u github.com/flarehotspot/go-utils@vx.x.x
```

## Building `dev-kit`

Change owner to `$USER` the `flarehotspot` dir first:
```shell
sudo chown -R $USER <flarehotspot dir>
```

then, `make devkit`.
