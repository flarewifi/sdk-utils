package boot

import (
	"core/internal/accounts"
	"fmt"
)

func InitAccounts() {
	fmt.Println("Initializing accounts...")
	accounts.EnsureAdminAcct()
}
