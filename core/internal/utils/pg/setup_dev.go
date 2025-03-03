//go:build dev

package pg

import "fmt"

func SetupServer(dbpass string, dbname string) error {
	fmt.Println("Skipping postgres server setup on dev mode...")
	return nil
}
