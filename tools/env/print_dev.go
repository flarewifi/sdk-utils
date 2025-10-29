//go:build dev

package env

import (
	"log"
)

func Print() {
	log.Println(lineComment)
	log.Println("Mode: ", "Development")
	log.Println("Http Port: ", HTTP_PORT)
	log.Println("RPC Token: ", RPC_TOKEN)
	log.Println(lineComment)
}
