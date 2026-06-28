package adminctrl

import (
	"core/internal/api"
	"core/internal/modules/updates"
	updatesview "core/resources/views/admin/updates"
	"core/utils/config"
	"core/utils/markdown"
	"core/utils/tags"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	sdkapi "sdk/api"
	"sync/atomic"
	"time"

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
		// The upload control accepts both firmware images and software-release archives;
		// advertise the broader set in the file picker (routing is content-based).
		allowedExts := updates.GetAcceptedUploadExtensions()
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

		// Staging paused: one or more plugins failed to build and we are waiting for
		// the admin to continue (skip them) or cancel. Render the decision dialog in
		// place of the progress bar; the surrounding poll keeps this live.
		if updates.AwaitingPluginConfirm() {
			page := updatesview.PluginBuildFailedPartial(api, updates.FailedPluginBuilds())
			if err := page.Render(r.Context(), w); err != nil {
				api.LoggerAPI.Error(err.Error())
			}
			return
		}

		// Staging paused: re-pinning a meta bundle would uninstall a member. Render the
		// abort/continue dialog in place of the progress bar; the poll keeps it live.
		if updates.AwaitingMetaRemovalConfirm() {
			page := updatesview.MetaMemberRemovalPartial(api, updates.RemovedMetaMembers())
			if err := page.Render(r.Context(), w); err != nil {
				api.LoggerAPI.Error(err.Error())
			}
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

// DownloadContinueCtrl resolves the plugin-build-failure dialog with "continue":
// the paused staging goroutine resumes, skips the failed plugins, and commits the
// rest. It returns no body — the download page's 1s poll picks up the resumed state
// and drives the flow to completion (download-done, or a live-applied success), so
// there is no redirect here that could re-trigger the download.
func DownloadContinueCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Resolve whichever gate is armed — build-failure or meta-member-removal. The
		// inactive one's Resolve is a no-op, so calling both is safe.
		updates.ResolvePluginFailureDecision(true)
		updates.ResolveMetaRemovalDecision(true)
		w.WriteHeader(http.StatusOK)
	}
}

// DownloadCancelCtrl resolves the plugin-build-failure dialog with "cancel": the
// staging goroutine discards the whole staged set and exits quietly (no update is
// applied, not even the core). It redirects back to the updates page with an info
// flash so the polling page stops and the admin lands somewhere sensible.
func DownloadCancelCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		// Resolve whichever gate is armed (the inactive one's Resolve is a no-op).
		updates.ResolvePluginFailureDecision(false)
		updates.ResolveMetaRemovalDecision(false)
		// Wait for the staging goroutine to finish unwinding (clear the in-flight flag
		// and discard the staged set) before redirecting. Otherwise the index page sees
		// IsDownloading()==true and bounces back to the progress page — the "stuck in
		// Compiling Plugins" symptom. Cleanup is fast (discard the staged dir); the cap
		// just bounds a pathological case.
		updates.WaitForStagingToStop(10 * time.Second)
		res.FlashMsg(w, r, api.Translate("info", "Software update cancelled"), sdkapi.FlashMsgInfo)
		res.Redirect(w, r, "admin:updates:index")
	}
}

// SysupgradeUploadCtrl handles the manual update upload. The same control accepts two
// kinds of file and routes by CONTENT, not by the filename the browser sent:
//   - a raw OS firmware image (.bin/.img) → the sysupgrade flash flow, or
//   - a software-release tarball (software-release[-mono]_*.tar.gz) → staged for
//     apply-on-reboot.
//
// A software-release tarball is always gzip-compressed; a firmware image is not, so a
// cheap 2-byte gzip probe is the first discriminator. A non-gzip upload takes the
// firmware path unchanged (byte-identical to the original behavior).
func SysupgradeUploadCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		// maxMemory only sets the in-RAM threshold; larger parts spill to disk. Keep it
		// modest so a 300 MB release isn't buffered in memory.
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			errMsg := api.Translate("error", "Failed to parse upload form")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		file, header, err := r.FormFile("sysupgrade_file")
		if err != nil {
			errMsg := api.Translate("error", "No file uploaded")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}
		defer file.Close()

		// Not gzip → a raw firmware image; run the existing sysupgrade flow directly off
		// the upload stream (no temp file, no extra copy).
		isGzip, _ := updates.IsGzip(file)
		if !isGzip {
			handleSysupgradeUpload(g, w, r, file, header.Filename, header.Size)
			return
		}

		handleSoftwareReleaseUpload(g, w, r, file, header)
	}
}

// handleSysupgradeUpload validates and saves a raw firmware image read from src, then
// runs the shared finalize (sysupgrade -T compatibility test + completion marker).
// src must be positioned at the start of the upload.
func handleSysupgradeUpload(g *api.CoreGlobals, w http.ResponseWriter, r *http.Request, src io.Reader, filename string, size int64) {
	api := g.CoreAPI
	res := api.HttpAPI.Response()

	if err := updates.ValidateSysupgradeFile(filename, size); err != nil {
		var errMsg string
		switch err {
		case updates.ErrInvalidFileExtension:
			errMsg = api.Translate("error", "Invalid file type. Upload a .bin or .img firmware image, or a software release archive") + "."
		case updates.ErrFileTooLarge:
			errMsg = api.Translate("error", "File size exceeds maximum allowed limit") + "."
		default:
			errMsg = api.Translate("error", "File validation failed")
		}
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}

	if err := updates.SaveSysupgradeFile(src, filename); err != nil {
		errMsg := api.Translate("error", "Failed to save firmware file")
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}

	// Validate firmware compatibility and create completion marker. This is the shared
	// path for both local uploads and remote downloads.
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

	successMsg := api.Translate("success", "Firmware uploaded and validated successfully")
	res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
	res.Redirect(w, r, "admin:updates:sysupgrade-success")
}

// handleSoftwareReleaseUpload spools a gzip upload to a temp file, confirms it is a
// flarewifi software-release tarball whose build flavor matches this machine, then
// stages it for apply-on-reboot. A gzip upload that is NOT a recognized release falls
// back to the firmware path — some firmware images are gzip-compressed, so a missed
// guess should still be validated by sysupgrade -T rather than rejected outright.
func handleSoftwareReleaseUpload(g *api.CoreGlobals, w http.ResponseWriter, r *http.Request, file multipart.File, header *multipart.FileHeader) {
	api := g.CoreAPI
	res := api.HttpAPI.Response()

	if header.Size > updates.MaxSoftwareReleaseFileSize {
		errMsg := api.Translate("error", "File size exceeds maximum allowed limit") + "."
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}

	tmpPath, err := updates.SaveUploadToTemp(file)
	if err != nil {
		errMsg := api.Translate("error", "Failed to save uploaded file")
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}
	defer os.Remove(tmpPath)

	info, err := updates.InspectRelease(tmpPath)
	if err != nil || !info.IsRelease {
		// gzip but not a recognized release — maybe a gzip-compressed firmware image.
		// Reopen the spooled file and let the firmware path validate it.
		f, oerr := os.Open(tmpPath)
		if oerr != nil {
			errMsg := api.Translate("error", "Failed to read uploaded file")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}
		defer f.Close()
		handleSysupgradeUpload(g, w, r, f, header.Filename, header.Size)
		return
	}

	// Reject a release built for the other app flavor: applying a mono whole-app
	// tarball on a non-mono machine (or vice versa) would brick it.
	if info.IsMono != tags.HasGoTag("mono") {
		errMsg := api.Translate("error", "This software release was built for a different system type and cannot be installed on this machine") + "."
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}

	if err := updates.StageLocalSoftwareRelease(g, tmpPath); err != nil {
		api.LoggerAPI.Error(fmt.Sprintf("stage local software release: %v", err))
		errMsg := api.Translate("error", "Failed to stage the uploaded software release") + "."
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}

	successMsg := api.Translate("success", "Software release uploaded and staged. Reboot to apply the update")
	res.FlashMsg(w, r, successMsg, sdkapi.FlashMsgSuccess)
	res.Redirect(w, r, "admin:updates:download-done")
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
