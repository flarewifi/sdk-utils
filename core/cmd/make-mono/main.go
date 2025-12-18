package main

import (
	"core/tools"
	"fmt"
)

func main() {
	fmt.Println("Generating mono files...")

	tools.CreateMonoPluginInit()
}
