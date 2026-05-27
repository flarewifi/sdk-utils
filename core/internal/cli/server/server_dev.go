//go:build !mono && dev

package server

import (
	"core/internal/api"
	"core/internal/boot"
)

func Server() {
	g := api.NewGlobals()
	defer g.Database.Close()

	boot.Init(g)
}
