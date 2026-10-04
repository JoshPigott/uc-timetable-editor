package handler

import (
	"errors"
	"net/http"
	"strconv"
)

type testConfirmation struct {
	Group   eventGroup
	Summary string
}

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
	updateEventGroupState(&group)
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

// ChangeTestState asks for confirmation only when a test type is being excluded.
func (h *Handler) ChangeTestState(w http.ResponseWriter, r *http.Request) {
	group, err := h.eventGroupFromRequest(w, r)
	if err != nil {
		http.Error(w, "Invalid event group request", http.StatusBadRequest)
		return
	}
	summary := r.FormValue("summary")
	if !groupHasTestType(group, summary) {
		http.Error(w, "Invalid test state request", http.StatusBadRequest)
		return
	}
	if isSummarySelected(group, summary) {
		h.renderEventGroup(w, "event-group-state-result", group)
		return
	}
	for index := range group.Options {
		if group.Options[index].Summary == summary {
			group.Options[index].Selected = true
			break
		}
	}
	updateEventGroupState(&group)
	h.renderTemplate(w, "confirm-test-state", testConfirmation{Group: group, Summary: summary})
}

// ApplyTestState excludes a test type after the user confirms the prompt.
func (h *Handler) ApplyTestState(w http.ResponseWriter, r *http.Request) {
	group, err := h.eventGroupFromRequest(w, r)
	if err != nil {
		http.Error(w, "Invalid event group request", http.StatusBadRequest)
		return
	}
	summary := r.FormValue("summary")
	if !groupHasTestType(group, summary) || !isSummarySelected(group, summary) {
		http.Error(w, "Invalid test confirmation request", http.StatusBadRequest)
		return
	}
	for index := range group.Options {
		if group.Options[index].Summary == summary {
			group.Options[index].Selected = false
			break
		}
	}
	updateEventGroupState(&group)
	h.renderEventGroup(w, "event-group-confirm-result", group)
}

// ConfirmEventGroupToggle asks before turning off a group that includes tests.
func (h *Handler) ConfirmEventGroupToggle(w http.ResponseWriter, r *http.Request) {
	group, err := h.eventGroupFromRequest(w, r)
	if err != nil {
		http.Error(w, "Invalid event group request", http.StatusBadRequest)
		return
	}
	if !group.AllSelected || len(group.SelectedTestTypes) == 0 {
		http.Error(w, "Invalid test confirmation request", http.StatusBadRequest)
		return
	}
	h.renderEventGroup(w, "confirm-event-group-toggle", group)
}

// CancelTestConfirmation closes a prompt without changing the selected event types.
func (h *Handler) CancelTestConfirmation(w http.ResponseWriter, r *http.Request) {
	if _, err := h.eventGroupFromRequest(w, r); err != nil {
		http.Error(w, "Invalid event group request", http.StatusBadRequest)
		return
	}
	h.renderTemplate(w, "dismiss-test-confirmation", nil)
}

func groupHasTestType(group eventGroup, summary string) bool {
	for _, option := range group.Options {
		if option.Summary == summary && option.IsTest {
			return true
		}
	}
	return false
}

func isSummarySelected(group eventGroup, summary string) bool {
	for _, option := range group.Options {
		if option.Summary == summary {
			return option.Selected
		}
	}
	return false
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
	groups := buildEventGroups(types, selectedSummaries(r.Form["summary_type"]), parseGroupingHints(r.FormValue("available_context")))
	if index >= len(groups) {
		return eventGroup{}, errors.New("event group does not exist")
	}
	return groups[index], nil
}

func (h *Handler) renderEventGroup(w http.ResponseWriter, name string, group eventGroup) {
	h.renderTemplate(w, name, group)
}

func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Could not update event filters", http.StatusInternalServerError)
	}
}
