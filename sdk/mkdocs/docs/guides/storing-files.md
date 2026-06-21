# Storing Files

This guide shows you how to store and manage files in your Flarewifi plugin using the Storage API.

## Overview

The Storage API allows plugins to store binary files like images, documents, and other assets. Files are:

- Stored in `data/storage/plugins/{your-plugin-package}/`
- Automatically accessible via HTTP
- Isolated per plugin (each plugin has its own storage directory)
- Size-limited to prevent abuse (10MB default, configurable)

## Common Use Cases

### Uploading Images

Handle image uploads from admin forms. Use [`api.Storage().WriteReader`](../api/storage-api.md#writereader) to save the uploaded stream, [`api.Http().Response().FlashMsg`](../api/http-response.md#flashmsg) for user feedback, and [`api.Http().Response().Redirect`](../api/http-response.md#redirect) to return to the form:

```go
package handlers

import (
    "net/http"
    sdkapi "sdk/api"
)

func UploadBrandingHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Get uploaded file
        file, header, err := r.FormFile("logo")
        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusBadRequest)
            return
        }
        defer file.Close()
        
        // Save to storage
        filename := "branding/logo.png"
        _, err = api.Storage().WriteReader(filename, file)
        if err != nil {
            api.Http().Response().FlashMsg(w, r, 
                api.Translate("error", "File upload failed"), "error")
            api.Http().Response().Redirect(w, r, "admin:settings:index")
            return
        }
        
        api.Http().Response().FlashMsg(w, r, 
            api.Translate("success", "Logo uploaded successfully"), "success")
        api.Http().Response().Redirect(w, r, "admin:settings:index")
    }
}
```

### Displaying Stored Images

Use stored images in your templ templates:

```templ
package views

import "sdk/api"

templ BrandingPage(api api.IPluginApi) {
    <div class="branding-preview">
        @if api.Storage().Exists("branding/logo.png") {
            <img 
                src={ templ.SafeURL(api.Storage().UrlFor("branding/logo.png")) } 
                alt="Company Logo"
                class="img-responsive"
            />
        } else {
            <p>{ api.Translate("label", "No logo uploaded") }</p>
        }
    </div>
}
```

### Storing Configuration Files

Save JSON or other config files:

```go
import "encoding/json"

type ThemeSettings struct {
    PrimaryColor   string `json:"primary_color"`
    SecondaryColor string `json:"secondary_color"`
    FontFamily     string `json:"font_family"`
}

func SaveThemeSettings(api sdkapi.IPluginApi, settings ThemeSettings) error {
    // Marshal to JSON
    data, err := json.Marshal(settings)
    if err != nil {
        return err
    }
    
    // Store in file
    _, err = api.Storage().Write("config/theme.json", data)
    return err
}

func LoadThemeSettings(api sdkapi.IPluginApi) (ThemeSettings, error) {
    var settings ThemeSettings
    
    // Read from storage
    data, err := api.Storage().Read("config/theme.json")
    if err != nil {
        return settings, err
    }
    
    // Unmarshal JSON
    err = json.Unmarshal(data, &settings)
    return settings, err
}
```

### Managing File Collections

List and manage multiple files:

```go
func GetAllBranding(api sdkapi.IPluginApi) ([]string, error) {
    // Get all images in branding directory
    files, err := api.Storage().List("branding/*")
    if err != nil {
        return nil, err
    }
    
    // Convert to URLs
    urls := make([]string, len(files))
    for i, file := range files {
        urls[i] = api.Storage().UrlFor(file)
    }
    
    return urls, nil
}

func CleanupOldBranding(api sdkapi.IPluginApi) error {
    // List all files
    files, err := api.Storage().List("branding/*")
    if err != nil {
        return err
    }
    
    // Keep only the latest
    if len(files) > 1 {
        for i := 0; i < len(files)-1; i++ {
            if err := api.Storage().Delete(files[i]); err != nil {
                api.Logger().Errorf("Failed to delete %s: %v", files[i], err)
            }
        }
    }
    
    return nil
}
```

### Organizing Files

Use subdirectories to organize files:

```go
// Store in organized structure
api.Storage().Write("images/logos/company.png", logoData)
api.Storage().Write("images/backgrounds/hero.jpg", bgData)
api.Storage().Write("documents/terms.pdf", termsData)
api.Storage().Write("config/settings.json", configData)

// List by category
logos, _ := api.Storage().List("images/logos/*")
backgrounds, _ := api.Storage().List("images/backgrounds/*")
documents, _ := api.Storage().List("documents/*")

// Recursive search
allImages, _ := api.Storage().List("images/**/*")
allPdfs, _ := api.Storage().List("**/*.pdf")
```

### Renaming and Moving Files

Reorganize files as needed:

```go
// Rename a file
err := api.Storage().Move("temp-logo.png", "branding/logo.png")

// Move between directories
err := api.Storage().Move("uploads/image.jpg", "gallery/photo-1.jpg")

// Rotate backups
err := api.Storage().Move("config/settings.json", "config/settings.backup.json")
err := api.Storage().Write("config/settings.json", newConfigData)
```

## Form Integration

Complete example with form handling:

```go
// routes.go
func AdminRoutes(api sdkapi.IPluginApi) {
    router := api.Http().Router()
    router.RegisterPost("admin:branding:upload", "/admin/branding/upload", 
        UploadBrandingHandler(api))
}

// handlers.go
func UploadBrandingHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Validate form
        if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
            api.Http().Response().Error(w, r, err, http.StatusBadRequest)
            return
        }
        
        // Get file
        file, header, err := r.FormFile("logo")
        if err != nil {
            api.Http().Response().FlashMsg(w, r, 
                api.Translate("error", "Please select a file"), "error")
            api.Http().Response().Redirect(w, r, "admin:branding:index")
            return
        }
        defer file.Close()
        
        // Validate file type (optional)
        contentType := header.Header.Get("Content-Type")
        if contentType != "image/png" && contentType != "image/jpeg" {
            api.Http().Response().FlashMsg(w, r, 
                api.Translate("error", "Only PNG and JPEG images allowed"), "error")
            api.Http().Response().Redirect(w, r, "admin:branding:index")
            return
        }
        
        // Save file
        _, err = api.Storage().WriteReader("branding/logo.png", file)
        if err != nil {
            api.Logger().Errorf("Failed to save logo: %v", err)
            api.Http().Response().FlashMsg(w, r, 
                api.Translate("error", "Failed to save file"), "error")
            api.Http().Response().Redirect(w, r, "admin:branding:index")
            return
        }
        
        api.Http().Response().FlashMsg(w, r, 
            api.Translate("success", "Logo uploaded successfully"), "success")
        api.Http().Response().Redirect(w, r, "admin:branding:index")
    }
}
```

## File Size Limits

Files are limited by the `PluginMaxFileSize` configuration (default: 10MB).

To check if a file will exceed limits before uploading:

```go
func ValidateFileSize(api sdkapi.IPluginApi, size int64) error {
    cfg, err := api.Config().Application().Get()
    if err != nil {
        return err
    }
    
    if size > cfg.PluginMaxFileSize {
        return fmt.Errorf("file size %d exceeds maximum %d", 
            size, cfg.PluginMaxFileSize)
    }
    
    return nil
}
```

## Error Handling

Always handle storage errors appropriately. Use [`api.Http().Response().Error`](../api/http-response.md#error) for fatal errors and [`api.Logger()`](../api/logger-api.md) to record details for debugging:

```go
_, err := api.Storage().Write("image.png", data)
if err != nil {
    if strings.Contains(err.Error(), "exceeds maximum allowed size") {
        // File too large
        api.Http().Response().FlashMsg(w, r, 
            api.Translate("error", "File is too large"), "error")
    } else if strings.Contains(err.Error(), "path traversal not allowed") {
        // Invalid filename
        api.Http().Response().FlashMsg(w, r, 
            api.Translate("error", "Invalid filename"), "error")
    } else {
        // Other error
        api.Logger().Errorf("Storage error: %v", err)
        api.Http().Response().Error(w, r, err, http.StatusInternalServerError)
    }
    return
}
```

## Best Practices

1. **Organize files** - Use subdirectories to group related files
2. **Validate uploads** - Check file types and sizes before storing
3. **Clean up** - Delete old/unused files to save disk space
4. **Error handling** - Always handle errors appropriately
5. **Security** - Never trust user-provided filenames, sanitize them
6. **Logging** - Log important storage operations for debugging

## Related

- [IStorageApi](../api/storage-api.md) — Complete storage API: `Write`, `WriteReader`, `Read`, `List`, `Delete`, `Move`, `Exists`, `UrlFor`
- [IHttpResponse](../api/http-response.md) — `Error`, `FlashMsg`, and `Redirect` methods used in upload handlers
- [ILoggerApi](../api/logger-api.md) — Logging storage errors with `Errorf`
- [IHttpFormsApi](../api/http-forms-api.md) — Handling `multipart/form-data` file uploads
- [Saving Data](./saving-data.md) — For structured plugin configuration data (`IConfigApi`)
