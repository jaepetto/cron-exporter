package dashboard

import (
	"embed"
	"encoding/json"
	"html/template"
	"io"
	"time"

	"github.com/jaepetto/cron-exporter/pkg/config"
)

//go:embed templates/*
var templatesFS embed.FS

// TemplateManager manages HTML templates for the dashboard
type TemplateManager struct {
	templates *template.Template
	config    *config.DashboardConfig
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(config *config.DashboardConfig) *TemplateManager {
	// Create function map for templates
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"timeAgo": func(t time.Time) string {
			duration := time.Since(t)
			if duration < time.Minute {
				return "just now"
			} else if duration < time.Hour {
				return formatDuration(duration, "minute")
			} else if duration < 24*time.Hour {
				return formatDuration(duration, "hour")
			} else {
				return formatDuration(duration, "day")
			}
		},
		"statusBadge": func(status string) string {
			switch status {
			case "active":
				return "success"
			case "maintenance":
				return "warning"
			case "paused":
				return "secondary"
			default:
				return "danger"
			}
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"marshalJSON": func(v interface{}) string {
			bytes, err := json.Marshal(v)
			if err != nil {
				return "{}"
			}
			return string(bytes)
		},
	}

	// Create template with functions
	tmpl := template.New("dashboard").Funcs(funcMap)

	// Parse embedded templates
	tmpl, err := tmpl.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		panic("Failed to parse dashboard templates: " + err.Error())
	}

	return &TemplateManager{
		templates: tmpl,
		config:    config,
	}
}

// LoadTemplates loads templates for Gin's HTML renderer
func LoadTemplates() *template.Template {
	// Create function map for templates
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"timeAgo": func(t time.Time) string {
			duration := time.Since(t)
			if duration < time.Minute {
				return "just now"
			} else if duration < time.Hour {
				return formatDuration(duration, "minute")
			} else if duration < 24*time.Hour {
				return formatDuration(duration, "hour")
			} else {
				return formatDuration(duration, "day")
			}
		},
		"statusBadge": func(status string) string {
			switch status {
			case "active":
				return "success"
			case "maintenance":
				return "warning"
			case "paused":
				return "secondary"
			default:
				return "danger"
			}
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
		"marshalJSON": func(v interface{}) string {
			bytes, err := json.Marshal(v)
			if err != nil {
				return "{}"
			}
			return string(bytes)
		},
	}

	// Create template with functions
	tmpl := template.New("").Funcs(funcMap)

	// Parse embedded templates
	tmpl, err := tmpl.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		panic("Failed to parse dashboard templates: " + err.Error())
	}

	return tmpl
}

// Render renders a template with the given data
func (tm *TemplateManager) Render(w io.Writer, name string, data interface{}) error {
	return tm.templates.ExecuteTemplate(w, name, data)
}

// RenderPartial renders a partial template for HTMX
func (tm *TemplateManager) RenderPartial(w io.Writer, name string, data interface{}) error {
	return tm.templates.ExecuteTemplate(w, name, data)
}

// formatDuration helper function for timeAgo
func formatDuration(d time.Duration, unit string) string {
	var value int64
	switch unit {
	case "minute":
		value = int64(d.Minutes())
	case "hour":
		value = int64(d.Hours())
	case "day":
		value = int64(d.Hours() / 24)
	}

	if value == 1 {
		return "1 " + unit + " ago"
	}
	return formatInt(value) + " " + unit + "s ago"
}

// formatInt formats an integer as string
func formatInt(i int64) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
