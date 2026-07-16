# IStorageApi

The `IStorageApi` provides file storage operations for plugins. Files are stored in `data/storage/plugins/{plugin-package}/` and are automatically accessible via HTTP.

## Overview

The Storage API allows plugins to:

- Store binary files (images, documents, etc.)
- Read and write files with automatic size validation
- Organize files in subdirectories
- Serve files via HTTP with automatic URLs
- Move/rename files while maintaining directory structure
- List and filter files using glob patterns
- Automatic cleanup of empty directories

## File Size Limits

By default, files are limited to 10MB. This can be configured in the application settings via `PluginMaxFileSize` field.

## Security

- All file paths are validated to prevent directory traversal attacks
- Each plugin can only access its own storage directory
- Files are stored with `0644` permissions (read/write for owner, read for others)

## Methods

### Write

Writes binary data to a file in the plugin's storage directory.

```go
data := []byte{...} // image bytes
path, err := api.Storage().Write("image.png", data)
if err != nil {
    // handle error (file too large, invalid path, etc.)
}
fmt.Println(path) // /path/to/data/storage/plugins/com.example.plugin/image.png
```

**Parameters:**

- `filename` (string): Relative path within plugin storage (supports subdirectories)
- `data` ([]byte): Binary data to write

**Returns:**

- Absolute filesystem path to the stored file
- Error if data exceeds `PluginMaxFileSize` or path is invalid

### Read

Reads binary data from a file in the plugin's storage directory.

```go
data, err := api.Storage().Read("image.png")
if err != nil {
    // handle error (file not found, permission denied, etc.)
}
// Use image data...
```

**Parameters:**

- `filename` (string): Relative path within plugin storage

**Returns:**

- File contents as []byte
- Error if file doesn't exist or can't be read

### WriteReader

Writes from an io.Reader to a file. Useful for HTTP file uploads.

```go
func UploadHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        file, header, err := r.FormFile("upload")
        if err != nil {
            // handle error
        }
        defer file.Close()
        
        path, err := api.Storage().WriteReader("uploads/" + header.Filename, file)
        if err != nil {
            // handle error (file too large, etc.)
        }
    }
}
```

**Parameters:**

- `filename` (string): Relative path within plugin storage
- `reader` (io.Reader): Data source

**Returns:**

- Absolute filesystem path
- Error if data exceeds size limit

### ReadReader

Returns an io.ReadCloser for streaming large files.

```go
reader, err := api.Storage().ReadReader("large-file.pdf")
if err != nil {
    // handle error
}
defer reader.Close()

// Stream to HTTP response
io.Copy(w, reader)
```

**Important:** Caller must close the reader when done.

### Delete

Removes a file and automatically cleans up empty parent directories.

```go
err := api.Storage().Delete("old-file.png")
if err != nil {
    // handle error
}
```

### Exists

Checks if a file exists in the plugin's storage directory.

```go
if api.Storage().Exists("config.json") {
    // file exists
} else {
    // file doesn't exist
}
```

### Move

Renames or moves a file within the plugin's storage directory. Overwrites destination if it exists.

```go
// Rename a file
err := api.Storage().Move("old-name.png", "new-name.png")

// Move to subdirectory
err := api.Storage().Move("temp.png", "images/final.png")

// Move and rename
err := api.Storage().Move("uploads/temp-123.jpg", "photos/vacation-2024.jpg")
```

Empty directories from the old location are automatically cleaned up.

### List

Returns filenames matching a glob pattern. Supports recursive patterns with `**`.

```go
// List all files
allFiles, err := api.Storage().List("")

// List all PNG files (any directory)
pngs, err := api.Storage().List("*.png")

// List all JPG files in images directory
images, err := api.Storage().List("images/*.jpg")

// Recursive: all PNGs in any subdirectory
allPngs, err := api.Storage().List("**/*.png")

// Pattern examples
configs, err := api.Storage().List("config.*")      // config.json, config.xml, etc.
thumbs, err := api.Storage().List("thumb_*.png")    // thumb_1.png, thumb_2.png, etc.
logs, err := api.Storage().List("2024-??-??.log")   // 2024-01-01.log, 2024-12-31.log, etc.
```

**Pattern Syntax:**

- `*` - matches any sequence of characters
- `?` - matches any single character
- `[range]` - matches character ranges (e.g., `[a-z]`, `[0-9]`)
- `**` - recursive directory matching
- Empty string - returns all files

### Path

Returns the absolute filesystem path for a filename.

```go
absPath := api.Storage().Path("image.png")
// Returns: /full/path/to/data/storage/plugins/com.example.plugin/image.png
```

### UrlFor

Returns the HTTP URL to access the stored file.

```go
url := api.Storage().UrlFor("logo.png")
// Returns: /storage/plugin/com.example.plugin/logo.png
```

Use in templates:

```templ
<img src={ templ.SafeURL(api.Storage().UrlFor("logo.png")) } alt="Logo" />
```

!!! note "The route is available even before your first write"
    Core registers the `/storage/plugin/<pkg>/` file-serving route only when the plugin's storage directory exists, and it pre-creates that directory for every plugin during initialization. So a file you upload at runtime is served immediately — you do **not** need to seed the directory yourself, and there is no "works only after a restart" gap. (Files still appear as they are written; `UrlFor` just returns the path — it does not create the file.)

## Complete Example

```go
package handlers

import (
    "net/http"
    sdkapi "sdk/api"
)

func UploadLogoHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Parse multipart form
        file, header, err := r.FormFile("logo")
        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusBadRequest)
            return
        }
        defer file.Close()
        
        // Store the file
        filename := "logos/" + header.Filename
        _, err = api.Storage().WriteReader(filename, file)
        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusInternalServerError)
            return
        }
        
        // Get URL to display
        logoURL := api.Storage().UrlFor(filename)
        
        // Redirect or render success
        api.Http().Response().FlashMsg(w, r, "Logo uploaded: " + logoURL, "success")
        api.Http().Response().Redirect(w, r, "admin:settings:index")
    }
}

func ListLogosHandler(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // List all logo files
        logos, err := api.Storage().List("logos/*.png")
        if err != nil {
            api.Http().Response().Error(w, r, err, http.StatusInternalServerError)
            return
        }
        
        // Build URLs
        urls := make([]string, len(logos))
        for i, logo := range logos {
            urls[i] = api.Storage().UrlFor(logo)
        }
        
        // Render view with logo URLs
        // ... render template
    }
}

func CleanupOldFiles(api sdkapi.IPluginApi) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // List all temporary files
        tempFiles, err := api.Storage().List("temp/*")
        if err != nil {
            // handle error
        }
        
        // Delete them
        for _, file := range tempFiles {
            if err := api.Storage().Delete(file); err != nil {
                api.Logger().Errorf("Failed to delete %s: %v", file, err)
            }
        }
        
        // Empty temp directory is automatically removed
    }
}
```

## See Also

- [Saving Data Guide](../guides/saving-data.md) - For plugin configuration data
- [HTTP Forms API](./http-forms-api.md) - For handling file uploads
- [Storing Files Guide](../guides/storing-files.md) - Complete guide with examples
