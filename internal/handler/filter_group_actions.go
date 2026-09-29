package handler

import (
	"errors"
	"net/http"
	"strconv"
)

// ToggleEventGroup applies the requested all-on or all-off state to a filter group.
func (h *Handler) ToggleEventGroup(w http.ResponseWriter, r *http.Request) {
	group, err := h.eventGroupFromRequest(w, r)
	if err != nil {
		http.Error(w, "Invalid event group request", http.StatusBadRequest)
		return
	}
	selectAll := !group.AllSelected
	for index := range group.Options {
		group.Options[index].Selected = selectAll
	}
	group.AllSelected = selectAll
	h.renderEventGroup(w, "event-group-toggle-result", group)
}

// UpdateEventGroupState refreshes the group button after an individual filter changes.
func (h *Handler) UpdateEventGroupState(w http.ResponseWriter, r *http.Request) {
	group, err := h.eventGroupFromRequest(w, r)
	if err != nil {
		http.Error(w, "Invalid event group request", http.StatusBadRequest)
		return
	}
	h.renderEventGroup(w, "event-group-state-result", group)
}

func (h *Handler) eventGroupFromRequest(w http.ResponseWriter, r *http.Request) (eventGroup, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		return eventGroup{}, err
	}
	index, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || index < 0 {
		return eventGroup{}, errors.New("invalid event group identifier")
	}
	types := cleanEventTypes(r.Form["available_summary"])
	groups := buildEventGroups(types, selectedSummaries(r.Form["summary_type"]))
	if index >= len(groups) {
		return eventGroup{}, errors.New("event group does not exist")
	}
	return groups[index], nil
}

func (h *Handler) renderEventGroup(w http.ResponseWriter, name string, group eventGroup) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := h.templates.ExecuteTemplate(w, name, group); err != nil {
		http.Error(w, "Could not update event filters", http.StatusInternalServerError)
	}
}
