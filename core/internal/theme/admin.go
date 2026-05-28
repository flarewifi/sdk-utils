package coretheme

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"

	corethemeadmin "core/resources/views/themes/fallback/admin"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// SetAdminTheme registers the built-in core admin theme on CoreAPI.
func SetAdminTheme(api sdkapi.IPluginApi) {
	api.Themes().NewAdminTheme(sdkapi.AdminThemeOpts{
		JsFile:  "theme-fallback.js",
		CssFile: "theme-fallback.css",
		CssLib:  sdkapi.CssLibBootstrap5,
		PreviewMeta: &sdkapi.ThemePreviewMeta{
			Background:     "#f9fafb",
			PrimaryColor:   "#2563eb",
			SecondaryColor: "#3b82f6",
			AccentColor:    "#60a5fa",
			ButtonColor:    "#2563eb",
			TextColor:      "#1f2937",
		},
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			notifs, err := api.Notification().GetUnreadNotifications(r.Context())
			if err != nil {
				notifs = []sdkapi.Notification{}
			}

			data := corethemeadmin.AdminLayoutData{
				Components:    c,
				Notifications: notifs,
				CurrentPath:   corethemeadmin.CurrentPathFromRequest(r),
			}
			layout := corethemeadmin.AdminLayout(api, data)
			if err := layout.Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering layout: %s</p>", err.Error())
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			fVersion := "1.0.0"
			if corePlugin, ok := api.PluginsMgr().FindByPkg("com.flarego.core"); ok {
				fVersion = corePlugin.Info().Version
			}

			info := getCoreSysInfo(api)

			page := corethemeadmin.AdminIndexPage(api, info, fVersion)
			return sdkapi.ViewPage{
				Assets:      sdkapi.ViewAssets{},
				PageContent: page,
			}
		},
	})
}

// getCoreSysInfo collects system metrics for the fallback dashboard.
func getCoreSysInfo(api sdkapi.IPluginApi) *corethemeadmin.SysInfo {
	info := &corethemeadmin.SysInfo{}

	// CPU usage
	cpuPercents, _ := cpu.Percent(0, true)
	info.CPUPercent = cpuPercents

	// Memory
	vmem, _ := mem.VirtualMemory()
	if vmem != nil {
		info.MemTotal = vmem.Total
		info.MemUsed = vmem.Used
		info.MemUsedPercent = vmem.UsedPercent
	}

	// Disk
	diskUsage, _ := disk.Usage("/")
	if diskUsage != nil {
		info.DiskTotal = diskUsage.Total
		info.DiskUsed = diskUsage.Used
		info.DiskUsedPercent = diskUsage.UsedPercent
	}

	// Uptime
	uptime, _ := host.Uptime()
	info.Uptime = uptime

	// Network
	iface, err := api.Network().GetWanInterface()
	if err == nil {
		if ipv4, err := iface.IpV4Addr(); err == nil {
			info.IPAddress = ipv4.Addr
		}
		if dev, err := iface.Device(); err == nil {
			info.MACAddress = dev.MacAddr()
		}
	}

	return info
}
