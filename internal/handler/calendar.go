package handler

import (
	"errors"
	"net/http"
	"strings"

	"timetable-editor/internal/backend"
)

// Calendar returns an iCal response for a saved feed token.
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" || len(token) > 100 {
		http.NotFound(w, r)
		return
	}
	calendar, err := h.service.Calendar(r.Context(), token)
	if errors.Is(err, backend.ErrFeedNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Calendar source unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(calendar)
}

// feedURL combines the configured public origin or this request's local origin with a token.
func (h *Handler) feedURL(r *http.Request, token string) string {
	base := h.publicBaseURL
	if base == "" {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		base = scheme + "://" + r.Host
	}
	return strings.TrimRight(base, "/") + "/feed/" + token + "/calendar.ics"
}
