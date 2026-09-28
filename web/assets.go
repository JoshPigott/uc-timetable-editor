package web

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed templates/*.html static/*
var assetFiles embed.FS

// ParseTemplates loads the page and HTMX response fragments from web/templates.
func ParseTemplates() (*template.Template, error) {
	return template.ParseFS(assetFiles, "templates/*.html")
}

// StaticFiles returns the embedded CSS and HTMX library directory.
func StaticFiles() fs.FS {
	static, err := fs.Sub(assetFiles, "static")
	if err != nil {
		panic(err)
	}
	return static
}
