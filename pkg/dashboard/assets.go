package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed assets/*
var assetsFS embed.FS

// AssetHandler serves embedded static assets
type AssetHandler struct {
	fileSystem http.FileSystem
}

// NewAssetHandler creates a new asset handler
func NewAssetHandler() *AssetHandler {
	// Create sub filesystem for assets directory
	sub, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		panic("Failed to create assets sub filesystem: " + err.Error())
	}

	return &AssetHandler{
		fileSystem: http.FS(sub),
	}
}

// ServeHTTP serves static assets
func (h *AssetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path
	cleanPath := path.Clean(r.URL.Path)

	// Remove leading slash
	if strings.HasPrefix(cleanPath, "/") {
		cleanPath = cleanPath[1:]
	}

	// Open the file
	file, err := h.fileSystem.Open(cleanPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set appropriate content type based on file extension
	contentType := getContentType(cleanPath)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	// Set caching headers for static assets
	w.Header().Set("Cache-Control", "public, max-age=86400") // 24 hours
	w.Header().Set("ETag", `"`+stat.ModTime().Format("20060102150405")+`"`)

	// Serve the file
	http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
}

// getContentType returns the appropriate content type for a file extension
func getContentType(filename string) string {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	default:
		return ""
	}
}
