package helpers

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"
)

func CurrentClient(r *http.Request) (sdkapi.IClientDevice, error) {
	clntSym := r.Context().Value(sdkapi.ClientCtxKey)
	if clntSym != nil {
		clnt, ok := clntSym.(sdkapi.IClientDevice)
		if ok {
			return clnt, nil
		}
	}

	return nil, fmt.Errorf("no client in context, make sure to use the device middleware")
}
