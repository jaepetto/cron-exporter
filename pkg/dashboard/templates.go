package dashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"regexp"
	"time"

	"github.com/jaepetto/cron-exporter/pkg/config"
	"github.com/jaepetto/cron-exporter/pkg/model"
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
		"highlightText": func(text, query string) template.HTML {
			if query == "" {
				return template.HTML(template.HTMLEscapeString(text))
			}
			return highlightTextHelper(text, query)
		},
		"buildSearchQuery": func(criteria interface{}, page int) string {
			return buildSearchQueryHelper(criteria, page)
		},
		"sequence": func(start, end int) []int {
			seq := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				seq = append(seq, i)
			}
			return seq
		},
		"add": func(a, b int) int {
			return a + b
		},
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"max": func(a, b int) int {
			if a > b {
				return a
			}
			return b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"eq": func(a, b interface{}) bool {
			return a == b
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

// highlightTextHelper highlights search terms in text
func highlightTextHelper(text, query string) template.HTML {
	if query == "" {
		return template.HTML(template.HTMLEscapeString(text))
	}

	// Escape the text first
	escaped := template.HTMLEscapeString(text)

	// Create case-insensitive regex for the query
	regex, err := regexp.Compile("(?i)" + regexp.QuoteMeta(query))
	if err != nil {
		return template.HTML(escaped)
	}

	// Replace matches with highlighted version
	highlighted := regex.ReplaceAllStringFunc(escaped, func(match string) string {
		return `<mark class="bg-warning">` + match + `</mark>`
	})

	return template.HTML(highlighted)
}

// buildSearchQueryHelper builds URL query string for pagination links
func buildSearchQueryHelper(criteria interface{}, page int) string {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))

	if crit, ok := criteria.(*model.JobSearchCriteria); ok && crit != nil {
		if crit.Query != "" {
			params.Set("q", crit.Query)
		}
		if crit.Name != "" {
			params.Set("name", crit.Name)
		}
		if crit.Host != "" {
			params.Set("host", crit.Host)
		}
		if crit.Status != "" {
			params.Set("status", crit.Status)
		}
		if crit.PageSize > 0 {
			params.Set("page_size", fmt.Sprintf("%d", crit.PageSize))
		}
	}

	return params.Encode()
}
