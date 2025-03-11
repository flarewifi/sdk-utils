package helpers

import (
	"net"
	"net/http"

	"core/internal/connmgr"
	"core/internal/utils/hostfinder"
	sdkapi "sdk/api"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CurrentClient(clntMgr *connmgr.ClientRegister, dbpool *pgxpool.Pool, r *http.Request) (sdkapi.IClientDevice, error) {
	clntSym := r.Context().Value(sdkapi.ClientCtxKey)
	if clntSym != nil {
		clnt, ok := clntSym.(sdkapi.IClientDevice)
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

	clnt, err := clntMgr.Register(dbpool, r, h.MacAddr, h.IpAddr, h.Hostname)
	if err != nil {
		return nil, err
	}

	return clnt, nil
}
