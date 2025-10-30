package controllers

import (
	"core/internal/api"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func GlobalAdminJsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	files := g.GlobalAssets.AdminJsFiles

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

		for i, file := range files {
			f, err := os.Open(file)
			if err != nil {
				http.Error(w, "Failed to open "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Optional: Write comment separators for clarity
			_, _ = io.WriteString(w, "\n/* ---- "+filepath.Base(file)+" ---- */\n")

			// Stream file content directly
			if _, err := io.Copy(w, f); err != nil {
				f.Close()
				http.Error(w, "Failed to stream "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}
			f.Close()

			// Add a semicolon and newline between scripts to avoid syntax issues
			if i < len(files)-1 {
				_, _ = io.WriteString(w, ";\n")
			}
		}
	}
}

func GlobalAdminCssCtrl(g *api.CoreGlobals) http.HandlerFunc {
	files := g.GlobalAssets.AdminCssFiles
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")

		for i, file := range files {
			f, err := os.Open(file)
			if err != nil {
				http.Error(w, "Failed to open "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Optional: Write comment separators for clarity
			_, _ = io.WriteString(w, "\n/* ---- "+filepath.Base(file)+" ---- */\n")

			// Stream file content directly
			if _, err := io.Copy(w, f); err != nil {
				f.Close()
				http.Error(w, "Failed to stream "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}
			f.Close()

			// Add a semicolon and newline between scripts to avoid syntax issues
			if i < len(files)-1 {
				_, _ = io.WriteString(w, ";\n")
			}
		}
	}
}

func GlobalPortalJsCtrl(g *api.CoreGlobals) http.HandlerFunc {
	files := g.GlobalAssets.PortalJsFiles

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

		for i, file := range files {
			f, err := os.Open(file)
			if err != nil {
				http.Error(w, "Failed to open "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Optional: Write comment separators for clarity
			_, _ = io.WriteString(w, "\n/* ---- "+filepath.Base(file)+" ---- */\n")

			// Stream file content directly
			if _, err := io.Copy(w, f); err != nil {
				f.Close()
				http.Error(w, "Failed to stream "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}
			f.Close()

			// Add a semicolon and newline between scripts to avoid syntax issues
			if i < len(files)-1 {
				_, _ = io.WriteString(w, ";\n")
			}
		}
	}
}
func GlobalPortalCssCtrl(g *api.CoreGlobals) http.HandlerFunc {
	files := g.GlobalAssets.PortalCssFiles

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")

		for i, file := range files {
			f, err := os.Open(file)
			if err != nil {
				http.Error(w, "Failed to open "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Optional: Write comment separators for clarity
			_, _ = io.WriteString(w, "\n/* ---- "+filepath.Base(file)+" ---- */\n")

			// Stream file content directly
			if _, err := io.Copy(w, f); err != nil {
				f.Close()
				http.Error(w, "Failed to stream "+file+": "+err.Error(), http.StatusInternalServerError)
				return
			}
			f.Close()

			// Add a semicolon and newline between scripts to avoid syntax issues
			if i < len(files)-1 {
				_, _ = io.WriteString(w, ";\n")
			}
		}
	}
}
