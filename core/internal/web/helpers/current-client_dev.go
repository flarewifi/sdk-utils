//go:build dev

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

	var ip, mac string

	macval, _ := r.Cookie("mac")
	ipval, _ := r.Cookie("ip")

	if macval != nil && macval.Value != "" {
		mac = macval.Value
	}

	if ipval != nil && ipval.Value != "" {
		ip = ipval.Value
	}

	if ip == "" {
		hostIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return nil, err
		}
		ip = hostIP
	}

	h, err := hostfinder.FindByIp(ip)
	if err != nil {
		return nil, err
	}

	if mac != "" {
		h.MacAddr = mac
	}

	clnt, err := clntMgr.Register(dbpool, r, h.MacAddr, h.IpAddr, h.Hostname)
	if err != nil {
		return nil, err
	}

	return clnt, nil
}
