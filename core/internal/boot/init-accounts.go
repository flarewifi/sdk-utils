package boot

import (
	"core/internal/accounts"
)

func InitAccounts() {
	accounts.EnsureAdminAcct()
}
