package handler

import (
	"html/template"

	"timetable-editor/internal/backend"
	"timetable-editor/web"
)

// Handler connects HTTP requests to the backend and renders web templates.
type Handler struct {
	service       *backend.Service
	publicBaseURL string
	templates     *template.Template
}

// New prepares the page templates and creates the HTTP handler set.
func New(service *backend.Service, publicBaseURL string) (*Handler, error) {
	templates, err := web.ParseTemplates()
	if err != nil {
		return nil, err
	}
	return &Handler{service: service, publicBaseURL: publicBaseURL, templates: templates}, nil
}

type pageData struct {
	SourceURL         string
	EventTypes        []string
	SelectedSummaries map[string]bool
	EventGroups       []eventGroup
	Analyzed          bool
	Error             string
	FeedURL           string
}
