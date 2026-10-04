package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"timetable-editor/internal/backend"
)

const maxFormBytes = 128 << 10

var labeledCoursePattern = regexp.MustCompile(`(?i)(?:course|paper|subject)(?:\s+title)?\s*[:=]\s*([^.;,\r\n]{3,80})`)

// AnalyzeCalendar reads the supplied calendar and returns its personalized event choices.
func (h *Handler) AnalyzeCalendar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		h.renderAnalyzeError(w, r, pageData{Error: "The form was too large or invalid."}, http.StatusBadRequest)
		return
	}
	sourceURL := strings.TrimSpace(r.FormValue("feed_url"))
	types, descriptions, err := h.service.EventTypesWithDescriptions(r.Context(), sourceURL)
	if err != nil {
		h.renderAnalyzeError(w, r, pageData{SourceURL: sourceURL, Error: err.Error()}, http.StatusBadRequest)
		return
	}
	selected := make(map[string]bool, len(types))
	for _, eventType := range types {
		selected[eventType] = true
	}
	hints := groupingHintsFromDescriptions(descriptions)
	hintsJSON, _ := json.Marshal(hints)
	data := pageData{
		SourceURL:         sourceURL,
		EventTypes:        types,
		SelectedSummaries: selected,
		Analyzed:          true,
		GroupingHints:     string(hintsJSON),
	}
	data.EventGroups = buildEventGroups(types, selected, hints)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if isHTMX(r) {
		if err := h.templates.ExecuteTemplate(w, "event-filters", data); err != nil {
			http.Error(w, "Could not display event filters", http.StatusInternalServerError)
		}
		return
	}
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Page unavailable", http.StatusInternalServerError)
	}
}

// CreateFeed validates form values and returns the new subscription link.
func (h *Handler) CreateFeed(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		h.renderFormError(w, r, pageData{Error: "The form was too large or invalid."}, http.StatusBadRequest)
		return
	}
	data := pageData{
		SourceURL:         strings.TrimSpace(r.FormValue("feed_url")),
		EventTypes:        cleanEventTypes(r.Form["available_summary"]),
		SelectedSummaries: selectedSummaries(r.Form["summary_type"]),
		Analyzed:          r.FormValue("event_types_ready") == "true",
		GroupingHints:     r.FormValue("available_context"),
	}
	hints := parseGroupingHints(data.GroupingHints)
	data.EventGroups = buildEventGroups(data.EventTypes, data.SelectedSummaries, hints)
	filters := backend.Filters{
		ExcludedSummaryTypes: excludedFromAvailable(data.EventTypes, data.SelectedSummaries),
		FilterSummary:        r.FormValue("filter_summary") == "true",
	}
	token, err := h.service.CreateFeed(r.Context(), data.SourceURL, filters)
	if err != nil {
		data.Error = err.Error()
		h.renderFormError(w, r, data, http.StatusBadRequest)
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

func groupingHintsFromDescriptions(descriptions map[string][]string) map[string][]string {
	hints := make(map[string][]string, len(descriptions))
	for summary, values := range descriptions {
		seen := make(map[string]bool)
		for _, description := range values {
			for _, match := range courseCodePattern.FindAllString(description, -1) {
				addGroupingHint(hints, summary, match, seen)
			}
			for _, parts := range labeledCoursePattern.FindAllStringSubmatch(description, -1) {
				course := strings.TrimSpace(parts[1])
				if len([]rune(course)) <= 80 {
					addGroupingHint(hints, summary, course, seen)
				}
			}
		}
	}
	return hints
}

func addGroupingHint(hints map[string][]string, summary, hint string, seen map[string]bool) {
	hint = strings.TrimSpace(hint)
	key := strings.ToLower(hint)
	if hint != "" && !seen[key] {
		hints[summary] = append(hints[summary], hint)
		seen[key] = true
	}
}

func parseGroupingHints(value string) map[string][]string {
	hints := make(map[string][]string)
	if value == "" || json.Unmarshal([]byte(value), &hints) != nil {
		return map[string][]string{}
	}
	return hints
}

func cleanEventTypes(values []string) []string {
	types := make([]string, 0, len(values))
	seen := make(map[string]bool)
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value != "" && !seen[key] {
			types = append(types, value)
			seen[key] = true
		}
	}
	return types
}

func selectedSummaries(values []string) map[string]bool {
	selected := make(map[string]bool)
	for _, value := range values {
		selected[strings.TrimSpace(value)] = true
	}
	return selected
}

func excludedFromAvailable(available []string, selected map[string]bool) []string {
	result := make([]string, 0, len(available))
	for _, eventType := range available {
		if !selected[eventType] {
			result = append(result, eventType)
		}
	}
	return result
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

func (h *Handler) renderAnalyzeError(w http.ResponseWriter, r *http.Request, data pageData, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if isHTMX(r) {
		w.WriteHeader(http.StatusOK)
		if err := h.templates.ExecuteTemplate(w, "form-error", data); err != nil {
			http.Error(w, "Could not display the calendar error", http.StatusInternalServerError)
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
