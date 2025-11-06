# AGENTS.md

## About this project

- This is a Go application that runs in OpenWRT routers
- It has two types of build:
  - With go plugins - it can install/uninstall plugins using native go "plugin" package
  - Monolithic build - all plugins are compiled as a single binary

## Build/Dev/Test

- `make` Runs the app with plugin install/uninstall capabilities, uses Go build tags "dev postgres"
- `make mono` Runs the monolithic app with Go build tags "dev mono sqlite"
- In production, we replace "dev" with "prod" Go build tag
- We don't implement automated tests - no unit test files

## Project Structure

- `go.work.default` - Copied to `go.work`, to be able to work on multiple Go modules
- `./scripts` - Scripts that need to run outside of Go context
- `./sdk/utils` Go utilities that can be reused in the core and plugins
- `./sdk/api` Go interfaces and structs API to build a plugin
- `./sdk/mkdocs` Documentation for the `sdk/api` usage
- `./core` The core of the system, it initializes the application and all the installed plugins
- `./core/internal/api` Contains the implementation of `./sdk/api`
- `./core/db` Contains the Go database queries generated from `./core/resources/queries/`
- `./core/resources/assets` Contains the javascript and css
- `./core/resources/views` Contains the `templ` files for our views
- `./core/internal/web` Contains routing, navigation, middlewares and controllers/handlers
- Each plugin has a corresponding `resources` directory similar to `./core/resources/`

## Tech Stack

- Using `Go` as primary programming language
- We are not allowed to exceed the go tool chain version defined in `go.work.default` when installing new libraries

- `docker compose` to run the app and database for easy development setup
- `gorilla/mux` for handling the routes
- `templ` for our views
- `sqlc` for our database queries
- `esbuild` Go API for bundling our assets
- `Makefile` To run common commands
