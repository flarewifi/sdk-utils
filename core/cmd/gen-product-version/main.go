// Command gen-product-version writes core/product.json with the version copied
// from core/plugin.json. It is run by the local-dev reflex build (start-dev.sh)
// so the machine has a product.json stand-in in dev, where the software-release
// stamp never runs. core/product.json is gitignored. See core/utils.GenProductVersion.
package main

import (
	"log"

	tools "core/utils"
)

func main() {
	if err := tools.GenProductVersion(); err != nil {
		log.Fatalf("gen-product-version: %v", err)
	}
}
