package admin

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	sdkutils "github.com/flarewifi/sdk-utils"
	sdkapi "sdk/api"

	"com.flarego.devkit/app/utils"
)

// maxUploadBytes caps a single plugin archive at 256 MiB — generous for source
// (which is text plus a few assets) while refusing a runaway upload outright.
const maxUploadBytes = 256 << 20

// UploadCtrl accepts a plugin archive (.zip / .tar.gz / .tar.xz), persists its
// source under data/plugins/local/<package> and installs it live. The install
// runs against the running core, so the plugin becomes available without a
// restart; the persisted source is loaded on the next boot without recompiling
// (local plugins are built once at install time, not on every boot).
func UploadCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()

		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			res.FlashMsg(w, r, api.Translate("error", "The upload was too large or could not be read"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		file, header, err := r.FormFile("plugin_archive")
		if err != nil {
			res.FlashMsg(w, r, api.Translate("error", "Please choose a plugin archive to upload"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}
		defer file.Close()

		// Scratch space for this one upload: the raw archive plus its extraction.
		// Removed before we return regardless of outcome — the source we keep is
		// the copy SaveSource places under data/plugins/local/.
		work := filepath.Join(sdkutils.PathTmpDir, "developer", "upload", sdkutils.RandomStr(12))
		if err := sdkutils.FsEnsureDir(work); err != nil {
			api.Logger().Error("developer: prepare upload workspace: " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Could not prepare the upload"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}
		defer os.RemoveAll(work)

		archivePath := filepath.Join(work, "archive")
		if err := saveUpload(file, archivePath); err != nil {
			api.Logger().Error("developer: save upload: " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Could not save the uploaded file"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		extractDir := filepath.Join(work, "src")
		if err := utils.ExtractArchive(archivePath, header.Filename, extractDir); err != nil {
			// Log the raw cause (it can carry a scratch path); show a generic reason.
			api.Logger().Error("developer: extract archive " + header.Filename + ": " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Could not extract the archive. Upload a valid .zip, .tar.gz or .tar.xz file"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		info, srcRoot, err := utils.FindAndValidateSrc(extractDir)
		if err != nil {
			res.FlashMsg(w, r, api.Translate("error", "Invalid plugin: <% .reason %>", "reason", err.Error()), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		// The core cannot reload a running plugin's .so in place, so refuse to let
		// the developer panel overwrite and reinstall itself.
		if info.Package == api.Info().Package {
			res.FlashMsg(w, r, api.Translate("error", "This plugin cannot install itself"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		localDest, err := utils.SaveSource(srcRoot, info.Package)
		if err != nil {
			api.Logger().Error("developer: save source for " + info.Package + ": " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Could not save the plugin source"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		// Install from the local copy. In a devkit build the core only permits
		// local installs whose source sits inside data/plugins/local/ (or the
		// in-tree data/plugins/devel/), which is exactly where SaveSource placed
		// it. The stripped (root-relative) path is what the core resolves and
		// confines against PathPluginLocalDir.
		handle, err := api.PluginsMgr().InstallPlugin(sdkutils.PluginSrcDef{
			Src:       sdkutils.PluginSrcLocal,
			LocalPath: sdkutils.StripRootPath(localDest),
		})
		if err != nil {
			api.Logger().Error("developer: start install for " + info.Package + ": " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "The plugin source was saved but the install could not be started"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		// Block until the background build/install settles so the developer gets a
		// definitive result on this request. A build can take a while; that is the
		// same trade-off the sysupgrade upload makes.
		if err := handle.Done(); err != nil {
			api.Logger().Error("developer: install " + info.Package + ": " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "The plugin source was saved but failed to build or install. Check the logs for details"), sdkapi.FlashMsgError)
			res.Redirect(w, r, "admin:developer:index")
			return
		}

		res.FlashMsg(w, r, api.Translate("success", "Plugin <% .name %> installed", "name", displayName(info)), sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, "admin:developer:index")
	}
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// saveUpload streams the multipart file to dst without buffering it all in memory.
func saveUpload(src io.Reader, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return err
	}
	return nil
}

// displayName prefers the plugin's declared name, falling back to its package id.
func displayName(info sdkutils.PluginInfo) string {
	if info.Name != "" {
		return info.Name
	}
	return info.Package
}
