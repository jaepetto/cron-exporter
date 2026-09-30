package dashboard

import (
	"embed"
	"fmt"
	"html"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaepetto/cron-exporter/pkg/config"
)

//go:embed web
var portalFS embed.FS

// PortalHandler serves the embedded React shell and its production assets.
type PortalHandler struct {
	fileSystem http.FileSystem
	indexHTML  []byte
}

// NewPortalHandler creates a portal handler with non-secret runtime values injected into the shell.
func NewPortalHandler(cfg *config.DashboardConfig) *PortalHandler {
	webFS, err := fs.Sub(portalFS, "web")
	if err != nil {
		panic("failed to create portal filesystem: " + err.Error())
	}

	indexHTML, err := fs.ReadFile(webFS, "index.html")
	if err != nil {
		indexHTML = []byte(`<!doctype html><html><head><meta charset="utf-8"><title>Portal build required</title></head><body><main><h1>Portal build required</h1><p>Run the frontend build before starting cronmetrics.</p></main></body></html>`)
	}
	basePath := html.EscapeString(cfg.Path)
	title := html.EscapeString(cfg.Title)
	indexHTML = []byte(strings.NewReplacer(
		"__CRONMETRICS_BASE_PATH__", basePath,
		"__CRONMETRICS_TITLE__", title,
	).Replace(string(indexHTML)))

	return &PortalHandler{fileSystem: http.FS(webFS), indexHTML: indexHTML}
}

// ServeIndex returns the current portal shell without cache retention.
func (h *PortalHandler) ServeIndex(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", h.indexHTML)
}

// ServeAsset returns a content-hashed production asset with immutable caching.
func (h *PortalHandler) ServeAsset(c *gin.Context) {
	assetPath := strings.TrimPrefix(path.Clean(c.Param("filepath")), "/")
	if assetPath == "." || strings.HasPrefix(assetPath, "../") {
		c.Status(http.StatusNotFound)
		return
	}

	file, err := h.fileSystem.Open(path.Join("assets", assetPath))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}
	contentType := mime.TypeByExtension(path.Ext(assetPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("ETag", fmt.Sprintf(`"%x-%x"`, info.Size(), info.ModTime().Unix()))
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
}
