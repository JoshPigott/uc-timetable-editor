package handler

import (
	"net/http"
	"strings"

	"timetable-editor/internal/backend"
)

const maxFormBytes = 16 << 10

// CreateFeed validates form values and returns the new subscription link.
func (h *Handler) CreateFeed(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, r, pageData{Error: "The form was too large or invalid."}, http.StatusBadRequest)
		return
	}
	data := pageData{SourceURL: strings.TrimSpace(r.FormValue("feed_url")), Terms: r.FormValue("terms"), Fields: selectedFields(r.Form["field"])}
	filters := backendFilters(r.FormValue("terms"), r.Form["field"])
	token, err := h.service.CreateFeed(r.Context(), data.SourceURL, filters)
	if err != nil {
		h.renderFormError(w, r, pageData{SourceURL: data.SourceURL, Terms: data.Terms, Fields: data.Fields, Error: err.Error()}, http.StatusBadRequest)
		return
	}
	data.FeedURL = h.feedURL(r, token)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if isHTMX(r) {
		if err := h.templates.ExecuteTemplate(w, "feed-result", data); err != nil {
			http.Error(w, "Could not display the feed URL", http.StatusInternalServerError)
		}
		return
	}
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Page unavailable", http.StatusInternalServerError)
	}
}

// selectedFields normalizes checked field names before backend validation.
func selectedFields(values []string) map[string]bool {
	selected := make(map[string]bool)
	for _, field := range values {
		selected[strings.ToLower(strings.TrimSpace(field))] = true
	}
	return selected
}

func backendFilters(termsText string, fields []string) backend.Filters {
	terms := strings.FieldsFunc(termsText, func(char rune) bool { return char == '\n' || char == '\r' })
	return backend.Filters{Terms: terms, Fields: fields}
}

func (h *Handler) renderFormError(w http.ResponseWriter, r *http.Request, data pageData, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		if err := h.templates.ExecuteTemplate(w, "form-error", data); err != nil {
			http.Error(w, "Could not display the form error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Page unavailable", http.StatusInternalServerError)
	}
}

func isHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}
