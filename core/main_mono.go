/**
WARNING: This file system-generated, do not edit nor commit into your repo.
Edit the main.go file instead to updated this file.
*/

//go:build mono

package core

import (
	"core/internal/api"
	"core/internal/boot"
)



func Init() {
	g := api.NewGlobals()
	defer g.Database.Close()

	boot.Init(g)
}
