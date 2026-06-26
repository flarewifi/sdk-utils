package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
	updatesview "core/resources/views/admin/updates"
	"core/utils/config"
	"core/utils/markdown"
	"errors"
	"fmt"
	"net/http"
	sdkapi "sdk/api"
	"sync/atomic"

	"github.com/a-h/templ"
)

const (
	EventInstallProgress = "software-update-progress"
)

var (
	newUpdate atomic.Value
)

// outdatedPlugins returns only the entries that have a newer version available,
// preserving order. Shared across builds (the list is empty on mono).
func outdatedPlugins(list []updates.PluginUpdate) []updates.PluginUpdate {
	out := make([]updates.PluginUpdate, 0, len(list))
	for _, p := range list {
		if p.HasUpdate {
			out = append(out, p)
		}
	}
	return out
}

// formatBytes converts bytes to human-readable format (B, KB, MB, GB)
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	if bytes < KB {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < MB {
		return fmt.Sprintf("%d KB", bytes/KB)
	} else if bytes < 100*MB {
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	} else if bytes < GB {
		return fmt.Sprintf("%d MB", bytes/MB)
	} else {
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	}
}

func CheckUpdatesPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		isDownloading := updates.IsDownloading()
		isDownloaded := updates.IsDownloaded()
		sysupgradeReady := updates.IsSysupgradeReady()

		if isDownloaded {
			res.Redirect(w, r, "admin:updates:download-done")
			return
		}
		if isDownloading {
			res.Redirect(w, r, "admin:updates:download")
			return
		}
		if sysupgradeReady {
			res.Redirect(w, r, "admin:updates:sysupgrade-success")
			return
		}

		cfg, err := config.ReadApplicationConfig()
		channel := "stable"
		if err == nil && cfg.Channel != "" {
			channel = cfg.Channel
		}

		newUpdate.Store(&updates.SoftwareReleaseUpdate{HasUpdate: false})
		uploadUrl := api.HttpAPI.Helpers().UrlForRoute("admin:updates:sysupgrade-upload")
		csrfHTML := api.HttpAPI.Helpers().CsrfHtmlTag(r)
		maxSizeMB := updates.GetMaxFileSizeMB()
		allowedExts := updates.GetAllowedExtensions()
		// Display the machine's product version (what it reports for updates), not
		// the core version.
		currentVersion := api.Machine().ProductVersion()
		page := updatesview.SoftwareUpdatesPage(api, channel, nil, false, uploadUrl, csrfHTML, maxSizeMB, allowedExts, currentVersion)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
			Assets: sdkapi.ViewAssets{
				JsFile: "check-updates.js",
			},
		})
	}
}

func QuerySoftwareUpdatesCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channel := r.FormValue("channel")
		if channel == "" {
			channel = "stable"
		}
		cfg2, err2 := config.ReadApplicationConfig()
		if err2 == nil {
			cfg2.Channel = channel
			_ = config.WriteApplicationConfig(cfg2)
		}

		api := g.CoreAPI

		// The machine's current version for both the update comparison (server-side)
		// and display is its PRODUCT version (per-partner lineage), not the core
		// version. CheckSoftwareReleaseUpdate sources it internally.
		currentVersion := api.Machine().ProductVersion()

		checkUpdateErr := errors.New(g.CoreAPI.Translate("error", "Unable to Check Updates"))
		result, err := updates.CheckSoftwareReleaseUpdate()
		if err != nil {
			page := updatesview.CheckForUpdatesPartial(api, updatesview.SoftwareUpdate{}, false, checkUpdateErr)
			page.Render(r.Context(), w)
			return
		}

		newUpdate.Store(result)

		var update updatesview.SoftwareUpdate
		if result.HasUpdate {
			// Parse markdown release notes to HTML
			var releaseNotesHTML templ.Component
			if result.ReleaseNotes != "" {
				htmlContent, err := markdown.ParseMarkdown(result.ReleaseNotes)
				if err == nil {
					releaseNotesHTML = templ.Raw(htmlContent)
				} else {
					releaseNotesHTML = templ.Raw("<pre>" + result.ReleaseNotes + "</pre>")
				}
			}

			update = updatesview.SoftwareUpdate{
				HasUpdate:        true,
				NewVersion:       result.Version.String(),
				CurrentVersion:   currentVersion,
				ReleaseNotes:     result.ReleaseNotes,
				ReleaseNotesHTML: releaseNotesHTML,
				IsSysupgrade:     result.IsSysupgrade,
			}
		} else {
			// Software is up to date - set current version for display
			update = updatesview.SoftwareUpdate{
				HasUpdate:      false,
				CurrentVersion: currentVersion,
			}
		}

		// Fetch the plugin update status once (best-effort, nil on mono/err) and keep
		// only the plugins that are actually outdated — the page hides plugins until
		// a check and then lists ONLY the ones to be updated.
		outdated := outdatedPlugins(checkPluginUpdatesList(g))
		hasPlugin := len(outdated) > 0

		page := updatesview.CheckForUpdatesPartial(api, update, hasPlugin, nil)
		page.Render(r.Context(), w)

		// Non-mono builds also fill the plugin update list via an OOB swap in the
		// same response. No-op on mono (see updates-ctrl_plugins_mono.go).
		renderPluginUpdatesOOB(g, w, r, outdated)
	}
}

func DownloadUpdatePageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		v := newUpdate.Load()
		update, _ := v.(*updates.SoftwareReleaseUpdate)
		coreUpdate := update != nil && update.HasUpdate

		isDownloading := updates.IsDownloading()
		isDownloaded := updates.IsDownloaded()

		// With no core update and nothing already staging/staged, this is only a
		// valid upgrade if some plugin is outdated. Otherwise there is nothing to do.
		if !coreUpdate && !isDownloading && !isDownloaded && !hasPluginUpdates(g) {
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		var percent int32
		if isDownloading {
			percent = updates.DownloadPercent()
		}

		if isDownloaded {
			percent = 100
		}

		compiling := updates.CurrentPhase() == updates.PhaseCompiling
		page := updatesview.DownloadUpdatePage(api, EventInstallProgress, int(percent), compiling, nil)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})

		if !isDownloaded && !isDownloading {
			// Initiate the upgrade. A core update stages the core plus every
			// ABI-matched plugin (model A, mono downloads a single tarball) via the
			// build-tagged startSystemDownload. With no core update we stage only the
			// latest plugins against the unchanged current core (startPluginOnlyDownload).
			if coreUpdate {
				startSystemDownload(g, update)
			} else {
				startPluginOnlyDownload(g)
			}
		}
	}
}

func DownloadStatusPartialCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		if updates.IsDownloaded() {
			api.HttpAPI.Response().Redirect(w, r, "admin:updates:download-done")
			return
		}

		// Plugin-only upgrade that applied live (a meta-bundle bump) — nothing to
		// reboot for. Flash success and return to the updates page.
		if updates.PluginUpdateApplied() {
			res := api.HttpAPI.Response()
			res.FlashMsg(w, r, api.Translate("success", "Plugins updated successfully"), sdkapi.FlashMsgSuccess)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		percent := updates.DownloadPercent()
		err := updates.DownloadError()
		downloaded := updates.DownloadedBytes()
		totalSize := updates.TotalSizeBytes()

		downloadedStr := formatBytes(downloaded)
		totalSizeStr := formatBytes(totalSize)

		compiling := updates.CurrentPhase() == updates.PhaseCompiling
		page := updatesview.DownloadStatusPartial(api, int(percent), downloadedStr, totalSizeStr, compiling, err)
		if err := page.Render(r.Context(), w); err != nil {
			api.LoggerAPI.Error(err.Error())
		}
	}
}

func DownloadDoneCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		if !updates.IsDownloaded() {
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		// Check if this is a sysupgrade by looking for the sysupgrade file
		isSysupgrade := updates.IsSysupgradeReady()

		// If it's a sysupgrade, redirect to the unified success page
		if isSysupgrade {
			successMsg := api.Translate("success", "Firmware downloaded and verified successfully")
			res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
			res.Redirect(w, r, "admin:updates:sysupgrade-success")
			return
		}

		// For regular updates, show the download done page
		page := updatesview.DownloadDonePage(api)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func SysupgradeUploadCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		// Parse multipart form (100 MB max)
		if err := r.ParseMultipartForm(updates.MaxSysupgradeFileSize); err != nil {
			errMsg := api.Translate("error", "Failed to parse upload form")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		// Get uploaded file
		file, header, err := r.FormFile("sysupgrade_file")
		if err != nil {
			errMsg := api.Translate("error", "No file uploaded")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}
		defer file.Close()

		// Validate file
		if err := updates.ValidateSysupgradeFile(header.Filename, header.Size); err != nil {
			var errMsg string
			switch err {
			case updates.ErrInvalidFileExtension:
				errMsg = api.Translate("error", "Invalid file type. Only .bin and .img files are allowed") + "."
			case updates.ErrFileTooLarge:
				errMsg = api.Translate("error", "File size exceeds maximum allowed limit") + "."
			default:
				errMsg = api.Translate("error", "File validation failed")
			}
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		// Save the file
		if err := updates.SaveSysupgradeFile(file, header.Filename); err != nil {
			errMsg := api.Translate("error", "Failed to save firmware file")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		// Validate firmware compatibility and create completion marker
		// This is the shared path for both local uploads and remote downloads
		if err := updates.FinalizeSysupgrade(); err != nil {
			var errMsg string
			switch err {
			case updates.ErrIncompatibleFirmware:
				errMsg = api.Translate("error", "The uploaded firmware is not compatible with this device") + "."
			default:
				errMsg = api.Translate("error", "Firmware validation failed")
			}
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		// Success - redirect to success page with options
		successMsg := api.Translate("success", "Firmware uploaded and validated successfully")
		res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin:updates:sysupgrade-success")
	}
}

func SysupgradeSuccessPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		// If sysupgrade is in progress, redirect to progress page
		if IsSysupgradeInProgress() {
			res.Redirect(w, r, "admin:updates:sysupgrade-progress")
			return
		}

		// Check if sysupgrade file exists, if not redirect to main updates page
		if !updates.IsSysupgradeReady() {
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		csrfHTML := api.HttpAPI.Helpers().CsrfHtmlTag(r)
		page := updatesview.SysupgradeSuccessPage(api, csrfHTML)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func SysupgradeProgressPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		page := updatesview.SysupgradeProgressPage(api)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func SysupgradeDeleteCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		// Remove the sysupgrade file and all related markers
		updates.RemoveSysupgradeFile()

		// Show success message and redirect to check for updates page
		successMsg := api.Translate("success", "Firmware file deleted successfully")
		res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin:updates:index")
	}
}
