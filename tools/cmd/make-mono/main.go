package main

import (
	"fmt"
	"tools"
)

func main() {
	fmt.Println("Generating mono files...")

	tools.CreateMonoPluginInit()
}
