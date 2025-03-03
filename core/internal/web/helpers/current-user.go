package helpers

// import (
// 	"errors"
// 	"net/http"

// 	"core/internal/accounts"
// 	"sdk/api/http"
// )

// func CurrentAcct(r *http.Request) (*accounts.Account, error) {
// 	sym := r.Context().Value(sdkhttp.SysAcctCtxKey)
// 	acct, ok := sym.(*accounts.Account)
// 	if !ok {
// 		return nil, errors.New("Can't determine current admin account.")
// 	}

// 	return acct, nil
// }
