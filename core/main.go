//go:build !mono

package main

import (
	"core/internal/api"
	"core/internal/boot"
)

func main() {}

func Init() {
	g := api.NewGlobals()
	boot.Init(g)
	defer g.Db.SqlDB().Close()
}
