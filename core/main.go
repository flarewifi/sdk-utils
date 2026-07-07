//go:build !mono

package main

import (
	"core/internal/api"
	"core/internal/boot"
)

func main() {}

func Init() {
	g := api.NewGlobals()
	defer g.Database.Close()

	// boot.Init blocks until a graceful shutdown signal is received (see its
	// doc comment) — it owns the process's whole lifetime, not just boot.
	boot.Init(g)
}
