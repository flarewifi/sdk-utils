package controllers

import (
	"fmt"
	"net/http"

	"core/internal/api"
	deviceview "core/resources/views/device"
	"core/utils/hostfinder"
)

func DeviceDiagCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h, err := hostfinder.GetHostFromRequest(r)
		if err != nil || h == nil || h.MacAddr == "" {
			http.Error(w, "Unable to identify device from network", http.StatusBadRequest)
			return
		}

		dev, err := g.Models.Device().FindByMac(ctx, h.MacAddr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Device not found for MAC %s", h.MacAddr), http.StatusNotFound)
			return
		}

		macs, err := g.Models.DeviceMac().FindByDeviceID(ctx, dev.ID())
		if err != nil {
			macs = nil
		}

		fingerprints, err := g.Models.DeviceFingerprint().FindByDeviceID(ctx, dev.ID())
		if err != nil {
			fingerprints = nil
		}

		params := deviceview.DeviceDiagParams{
			DeviceID:     dev.ID(),
			UUID:         dev.UUID(),
			CookieToken:  dev.CookieToken(),
			Ipv4Addr:     dev.Ipv4Addr(),
			Ipv6Addr:     dev.Ipv6Addr(),
			MacAddr:      dev.MacAddr(),
			Hostname:     dev.Hostname(),
			Status:       int(dev.Status()),
			CreatedAt:    dev.CreatedAt().UTC().Format("2006-01-02 15:04:05 UTC"),
			UpdatedAt:    dev.UpdatedAt().UTC().Format("2006-01-02 15:04:05 UTC"),
			Macs:         macs,
			Fingerprints: fingerprints,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		deviceview.DeviceDiagPage(params).Render(ctx, w)
	}
}
