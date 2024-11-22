package helpers

import (
	"net"
	"net/http"

	"core/internal/connmgr"
	"sdk/api/connmgr"
	"sdk/api/http"
	"core/internal/utils/hostfinder"
)

func CurrentClient(clntMgr *connmgr.ClientRegister, r *http.Request) (sdkconnmgr.IClientDevice, error) {
	clntSym := r.Context().Value(sdkhttp.ClientCtxKey)
	if clntSym != nil {
        clnt, ok := clntSym.(sdkconnmgr.IClientDevice)
        if ok {
            return clnt, nil
        }
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}

	h, err := hostfinder.FindByIp(ip)
	if err != nil {
		return nil, err
	}

	clnt, err := clntMgr.Register(r, h.MacAddr, h.IpAddr, h.Hostname)
	if err != nil {
		return nil, err
	}

	return clnt, nil
}
