package adminctrl

import (
	"bytes"
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

	"github.com/Masterminds/semver/v3"
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
		// same response, with the pending core update (if any) listed alongside the
		// plugin versions. No-op on mono (see updates-ctrl_plugins_mono.go).
		renderPluginUpdatesOOB(g, w, r, outdated, update.CurrentVersion, update.NewVersion)
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

		// Plugin-only upgrade that applied live (a meta-bundle bump) — nothing to
		// reboot for. Flash success and return to the updates page.
		if updates.PluginUpdateApplied() {
			res := api.HttpAPI.Response()
			res.FlashMsg(w, r, api.Translate("success", "Plugins updated successfully"), sdkapi.FlashMsgSuccess)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		// Staging stopped without landing in any of the states above — the only way
		// that happens is a cancelled run unwinding quietly (see ErrUpdateCancelled in
		// system-update.go: no error stored, no staged-complete marker). This page polls
		// unconditionally every 1s (htmx hx-trigger on download-updates.templ) with no
		// other way to detect "downloading stopped"; without this check a cancel that
		// only finishes after DownloadCancelCtrl's wait times out would poll this same
		// static partial forever. Send the admin back rather than leaving them stuck.
		if !updates.IsDownloading() {
			res := api.HttpAPI.Response()
			res.FlashMsg(w, r, api.Translate("info", "Software update cancelled"), sdkapi.FlashMsgInfo)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		percent := updates.DownloadPercent()
		err := updates.DownloadError()

		compiling := updates.CurrentPhase() == updates.PhaseCompiling
		page := updatesview.DownloadStatusPartial(api, int(percent), compiling, err)
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

		// For regular updates, show the download done page with a summary of what
		// this run actually changed: the core version delta (if any — newUpdate still
		// holds the check result that kicked off this download, see
		// DownloadUpdatePageCtrl) plus the plugins staged/skipped during it.
		v := newUpdate.Load()
		update, _ := v.(*updates.SoftwareReleaseUpdate)
		coreUpdated := update != nil && update.HasUpdate
		fromVersion, toVersion := "", ""
		if coreUpdated {
			fromVersion = api.Machine().ProductVersion()
			if update.Version != nil {
				toVersion = update.Version.String()
			}
		}

		page := updatesview.DownloadDonePage(api, coreUpdated, fromVersion, toVersion, updates.LastStagedPlugins(), updates.LastSkippedPlugins())
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
			Assets: sdkapi.ViewAssets{
				JsFile: "reboot-wait.js",
			},
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
		updates.ResolvePluginFailureDecision(true)
		w.WriteHeader(http.StatusOK)
	}
}

// DownloadCancelCtrl cancels an in-flight software update from either of the two
// places the admin can trigger it from on the download page: the ordinary
// download/compiling progress view, or the plugin-build-failure confirmation
// dialog. The staging goroutine discards whatever it staged and exits quietly (no
// update is applied, not even the core). If staging is still unwinding a build that
// was already in flight when cancel was requested, this redirects to the progress
// page instead of claiming success — see the IsDownloading branch below and
// DownloadStatusPartialCtrl's matching check.
func DownloadCancelCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()
		// Only one of these two is ever actually relevant at a time (the gate pauses
		// the goroutine, so it can't also be mid-download) — calling both unconditionally
		// keeps this one handler correct for either pause point without the caller
		// needing to know which one is active. Both are no-ops when not applicable.
		updates.ResolvePluginFailureDecision(false)
		updates.RequestCancelDownload()
		// Cancellation is cooperative: it is only checked BETWEEN plugin builds, never
		// mid-build (see cancelRequested in updates.go), so a local plugin's on-device
		// `go build -buildmode=plugin` already in flight always finishes first. On real
		// OpenWRT hardware that routinely takes longer than this wait. Give it a chance
		// to land quickly, but do NOT assume it did.
		updates.WaitForStagingToStop(10 * time.Second)

		if updates.IsDownloading() {
			// Still unwinding. Claiming "cancelled" here and redirecting to the index
			// would immediately bounce back to this same progress page anyway (see
			// CheckUpdatesPageCtrl's IsDownloading redirect) — the "stuck in Compiling
			// Plugins" symptom that has led admins to power-cycle the device thinking
			// cancel failed, killing an in-flight on-device plugin recompile outright
			// and corrupting it for the NEXT update attempt. Tell the truth instead:
			// go to the progress page, which keeps polling (DownloadStatusPartialCtrl
			// now detects the moment staging actually stops and redirects itself).
			res.FlashMsg(w, r, api.Translate("info", "Cancelling the update — finishing the current step, this may take a moment"), sdkapi.FlashMsgInfo)
			res.Redirect(w, r, "admin:updates:download")
			return
		}

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

		// Stream the multipart body part-by-part instead of ParseMultipartForm/FormFile,
		// which reads the WHOLE body up front (in memory up to its maxMemory threshold,
		// spilling only the excess to a temp file) before this handler ever runs. This
		// way the "sysupgrade_file" part is copied straight to its final destination —
		// the full upload never sits in RAM or an intermediate temp file.
		mr, err := r.MultipartReader()
		if err != nil {
			errMsg := api.Translate("error", "Failed to parse upload form")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		var part *multipart.Part
		for {
			p, perr := mr.NextPart()
			if perr == io.EOF {
				break
			}
			if perr != nil {
				errMsg := api.Translate("error", "Failed to parse upload form")
				res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
				res.Redirect(w, r, "admin:updates:index")
				return
			}
			if p.FormName() == "sysupgrade_file" {
				part = p
				break
			}
			p.Close()
		}

		if part == nil {
			errMsg := api.Translate("error", "No file uploaded")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}
		defer part.Close()

		filename := part.FileName()

		// Not gzip → a raw firmware image; run the existing sysupgrade flow directly off
		// the upload stream (no temp file, no extra copy). The multipart part is
		// forward-only (no Seek), so the gzip magic-byte peek can't rewind like IsGzip
		// does for a seekable reader — instead, the peeked bytes are re-prepended with
		// io.MultiReader so nothing is lost.
		magic := make([]byte, 2)
		n, _ := io.ReadFull(part, magic)
		isGzip := n == 2 && magic[0] == 0x1f && magic[1] == 0x8b
		prefixed := io.NopCloser(io.MultiReader(bytes.NewReader(magic[:n]), part))

		if !isGzip {
			// The size cap is enforced DURING the copy (see SaveSysupgradeFile), since a
			// streamed upload's size isn't known until it's fully read.
			src := http.MaxBytesReader(w, prefixed, updates.MaxSysupgradeFileSize)
			handleSysupgradeUpload(g, w, r, src, filename)
			return
		}

		src := http.MaxBytesReader(w, prefixed, updates.MaxSoftwareReleaseFileSize)
		handleSoftwareReleaseUpload(g, w, r, src, filename)
	}
}

// handleSysupgradeUpload validates and saves a raw firmware image streamed from src,
// then runs the shared finalize (sysupgrade -T compatibility test + completion
// marker). src must be positioned at the start of the upload.
func handleSysupgradeUpload(g *api.CoreGlobals, w http.ResponseWriter, r *http.Request, src io.Reader, filename string) {
	api := g.CoreAPI
	res := api.HttpAPI.Response()

	if err := updates.ValidateSysupgradeExtension(filename); err != nil {
		errMsg := api.Translate("error", "Invalid file type. Upload a .bin or .img firmware image, or a software release archive") + "."
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}

	if err := updates.SaveSysupgradeFile(src, filename); err != nil {
		var errMsg string
		switch err {
		case updates.ErrFileTooLarge:
			errMsg = api.Translate("error", "File size exceeds maximum allowed limit") + "."
		default:
			errMsg = api.Translate("error", "Failed to save firmware file")
		}
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
// src's size is capped by the caller (http.MaxBytesReader at MaxSoftwareReleaseFileSize)
// since a streamed upload's size isn't known upfront.
func handleSoftwareReleaseUpload(g *api.CoreGlobals, w http.ResponseWriter, r *http.Request, src io.Reader, filename string) {
	api := g.CoreAPI
	res := api.HttpAPI.Response()

	tmpPath, err := updates.SaveUploadToTemp(src)
	if err != nil {
		var errMsg string
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			errMsg = api.Translate("error", "File size exceeds maximum allowed limit") + "."
		} else {
			errMsg = api.Translate("error", "Failed to save uploaded file")
		}
		res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
		res.Redirect(w, r, "admin:updates:index")
		return
	}
	defer os.Remove(tmpPath)

	info, err := updates.InspectRelease(tmpPath)
	if err != nil || !info.IsRelease {
		// gzip but not a recognized release — maybe a gzip-compressed firmware image.
		// It was spooled under the (larger) release cap, so re-check its actual on-disk
		// size against the firmware cap before treating it as one.
		st, serr := os.Stat(tmpPath)
		if serr != nil {
			errMsg := api.Translate("error", "Failed to read uploaded file")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}
		if st.Size() > updates.MaxSysupgradeFileSize {
			errMsg := api.Translate("error", "File size exceeds maximum allowed limit") + "."
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		f, oerr := os.Open(tmpPath)
		if oerr != nil {
			errMsg := api.Translate("error", "Failed to read uploaded file")
			res.FlashMsg(w, r, errMsg, sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:updates:index")
			return
		}
		defer f.Close()
		handleSysupgradeUpload(g, w, r, f, filename)
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

	// A manually uploaded release always carries a core payload (checked above via
	// HasPendingCoreUpdate in StageLocalSoftwareRelease), but this path never goes
	// through QuerySoftwareUpdatesCtrl, which is the only other place that populates
	// newUpdate. Store it here too so DownloadDoneCtrl's existing coreUpdated/
	// fromVersion/toVersion logic (shared with the online check-and-download flow)
	// lists "System core" for this path as well, instead of silently omitting it.
	targetVersion := info.ProductVersion
	if targetVersion == "" {
		// A manually uploaded release predating the mandatory core/product.json stamp
		// (writeProductVersion) may ship without one — fall back to the core ABI
		// version, mirroring product.Version()'s own fallback.
		targetVersion = info.CoreVersion
	}
	parsedVersion, _ := semver.NewVersion(targetVersion)
	newUpdate.Store(&updates.SoftwareReleaseUpdate{HasUpdate: true, Version: parsedVersion})

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
			Assets: sdkapi.ViewAssets{
				JsFile: "reboot-wait.js",
			},
		})
	}
}

// SysupgradeStatusPartialCtrl reports the live state of an in-flight sysupgrade so the
// progress page's poll can flip from the spinner to a failure state. There is no
// "done" branch: a successful sysupgrade reboots the device before this is ever
// polled again, so the only outcomes worth rendering are "still running" or "failed".
func SysupgradeStatusPartialCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		page := updatesview.SysupgradeProgressPartial(api, SysupgradeError())
		if renderErr := page.Render(r.Context(), w); renderErr != nil {
			api.LoggerAPI.Error(renderErr.Error())
		}
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
