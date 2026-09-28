package handler

import (
	"net/http"
)

// Home renders the calendar filter form.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	data := pageData{Fields: map[string]bool{"summary": true}}
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Page unavailable", http.StatusInternalServerError)
	}
}
