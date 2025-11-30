package adminctrl

import (
	"core/internal/api"
	"core/internal/utils/markdown"
	"core/internal/utils/updates"
	updatesview "core/resources/views/admin/updates"
	"errors"
	"log"
	"net/http"
	sdkapi "sdk/api"
	"sync/atomic"
	"tools/config"

	"github.com/Masterminds/semver/v3"
	"github.com/a-h/templ"
)

const (
	EventInstallProgress = "software-update-progress"
)

var (
	newUpdate atomic.Value
)

func CheckUpdatesPageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		isDownloading := updates.IsDownloading()
		isDownloaded := updates.IsDownloaded()
		if isDownloaded {
			res.Redirect(w, r, "admin:updates:download-done")
			return
		}
		if isDownloading {
			res.Redirect(w, r, "admin:updates:download")
			return
		}

		cfg, err := config.ReadApplicationConfig()
		channel := "stable"
		if err == nil && cfg.Channel != "" {
			channel = cfg.Channel
		}

		newUpdate.Store(&updates.SoftwareReleaseUpdate{HasUpdate: false})
		page := updatesview.SoftwareUpdatesPage(api, channel, nil)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
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
		coreInfo := api.Info()

		checkUpdateErr := errors.New(g.CoreAPI.Translate("error", "Unable to Check Updates"))
		currentVersion, err := semver.NewVersion(coreInfo.Version)
		if err != nil {
			log.Println("Error:", err)
			cfgFallback, _ := config.ReadApplicationConfig()
			channelFallback2 := "stable"
			if cfgFallback.Channel != "" {
				channelFallback2 = cfgFallback.Channel
			}
			page := updatesview.SoftwareUpdatesPage(api, channelFallback2, checkUpdateErr)
			page.Render(r.Context(), w)
			return
		}

		result, err := updates.CheckSoftwareReleaseUpdate(currentVersion)
		if err != nil {
			log.Println("Error:", err)
			cfg3, _ := config.ReadApplicationConfig()
			channelFallback := "stable"
			if cfg3.Channel != "" {
				channelFallback = cfg3.Channel
			}
			page := updatesview.SoftwareUpdatesPage(api, channelFallback, checkUpdateErr)
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
					// Fallback to plain text if parsing fails
					log.Println("Error parsing markdown:", err)
					releaseNotesHTML = templ.Raw("<pre>" + result.ReleaseNotes + "</pre>")
				}
			}

			update = updatesview.SoftwareUpdate{
				HasUpdate:        true,
				NewVersion:       result.Version.String(),
				CurrentVersion:   currentVersion.String(),
				ReleaseNotes:     result.ReleaseNotes,
				ReleaseNotesHTML: releaseNotesHTML,
			}
		}

		page := updatesview.CheckForUpdatesPartial(api, update, nil)
		page.Render(r.Context(), w)
	}
}

func DownloadUpdatePageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		v := newUpdate.Load()
		update, ok := v.(*updates.SoftwareReleaseUpdate)
		if !ok {
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		if !update.HasUpdate {
			res.Redirect(w, r, "admin:updates:index")
			return
		}

		isDownloading := updates.IsDownloading()
		isDownloaded := updates.IsDownloaded()

		var percent int32
		if isDownloading {
			percent = updates.DownloadPercent()
		}

		if isDownloaded {
			percent = 100
		}

		page := updatesview.DownloadUpdatePage(api, EventInstallProgress, int(percent), nil)
		res.AdminView(w, r, sdkapi.ViewPage{PageContent: page})

		if !isDownloaded && !isDownloading {
			// Initiate the process of downloading and installing of software updates
			go updates.DownloadSoftwareRelease(update.ReleseFileURL, update.ReleaseFileChecksum)
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

		percent := updates.DownloadPercent()
		err := updates.DownloadError()
		page := updatesview.DownloadStatusPartial(api, int(percent), err)
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

		page := updatesview.DownloadDonePage(api)
		res.AdminView(w, r, sdkapi.ViewPage{
			PageContent: page,
		})
	}
}

func SysupgradePageCtrl(g *api.CoreGlobals) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api := g.CoreAPI
		res := api.HttpAPI.Response()

		// If already downloaded, redirect to download-done page
		if updates.IsDownloaded() {
			res.Redirect(w, r, "admin:updates:download-done")
			return
		}

		uploadUrl := api.HttpAPI.Helpers().UrlForRoute("admin:updates:sysupgrade-upload")
		csrfHTML := api.HttpAPI.Helpers().CsrfHtmlTag(r)
		maxSizeMB := updates.GetMaxFileSizeMB()
		allowedExts := updates.GetAllowedExtensions()

		page := updatesview.SysupgradePage(api, uploadUrl, csrfHTML, maxSizeMB, allowedExts)
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
			log.Println("Error parsing multipart form:", err)
			errMsg := api.Translate("error", "Failed to parse upload form")
			page := updatesview.SysupgradeUploadError(api, errMsg)
			page.Render(r.Context(), w)
			return
		}

		// Get uploaded file
		file, header, err := r.FormFile("sysupgrade_file")
		if err != nil {
			log.Println("Error getting form file:", err)
			errMsg := api.Translate("error", "No file uploaded")
			page := updatesview.SysupgradeUploadError(api, errMsg)
			page.Render(r.Context(), w)
			return
		}
		defer file.Close()

		// Validate file
		if err := updates.ValidateSysupgradeFile(header.Filename, header.Size); err != nil {
			log.Println("File validation error:", err)
			var errMsg string
			switch err {
			case updates.ErrInvalidFileExtension:
				errMsg = api.Translate("error", "Invalid file type. Only .bin and .img files are allowed.")
			case updates.ErrFileTooLarge:
				errMsg = api.Translate("error", "File size exceeds maximum allowed limit.")
			default:
				errMsg = api.Translate("error", "File validation failed")
			}
			page := updatesview.SysupgradeUploadError(api, errMsg)
			page.Render(r.Context(), w)
			return
		}

		// Save the file
		if err := updates.SaveSysupgradeFile(file, header.Filename); err != nil {
			log.Println("Error saving sysupgrade file:", err)
			errMsg := api.Translate("error", "Failed to save firmware file")
			page := updatesview.SysupgradeUploadError(api, errMsg)
			page.Render(r.Context(), w)
			return
		}

		// Validate firmware compatibility with the device
		if err := updates.ValidateSysupgradeCompatibility(); err != nil {
			log.Println("Firmware compatibility check failed:", err)
			// Remove the incompatible firmware file
			updates.RemoveSysupgradeFile()
			var errMsg string
			switch err {
			case updates.ErrIncompatibleFirmware:
				errMsg = api.Translate("error", "The uploaded firmware is not compatible with this device.")
			default:
				errMsg = api.Translate("error", "Firmware validation failed")
			}
			page := updatesview.SysupgradeUploadError(api, errMsg)
			page.Render(r.Context(), w)
			return
		}

		// Success - redirect to download-done page
		res.FlashMsg(w, r, api.Translate("success", "Firmware uploaded and validated successfully"), sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin:updates:download-done")
	}
}
