/**
WARNING: This file system-generated, do not edit nor commit into your repo.
*/

//go:build mono

package core

import (
	"core/internal/api"
	"core/internal/boot"
)



func Init() {
	g := api.NewGlobals()
	defer g.Db.SqlDB().Close()

	boot.Init(g)
}
