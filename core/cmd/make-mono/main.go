package main

import (
	tools "core/utils"
	"fmt"
)

func main() {
	fmt.Println("Generating mono files...")

	tools.CreateMonoPluginInit()
}
