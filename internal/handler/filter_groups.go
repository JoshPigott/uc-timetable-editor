package handler

import (
	"regexp"
	"strconv"
	"strings"
)

type eventGroup struct {
	ID                string
	Name              string
	Options           []eventOption
	AllSelected       bool
	SelectedTestTypes []string
}

type eventOption struct {
	Summary  string
	Label    string
	Selected bool
	IsTest   bool
}

type eventGroupBuilder struct {
	name    string
	options []eventOption
}

var (
	colonCoursePattern   = regexp.MustCompile(`^([^:]{1,100}):\s*(.+)$`)
	numberCoursePattern  = regexp.MustCompile(`^(.{1,100}?\b\d{1,4}[A-Za-z]?)[\s,:-]+(.+)$`)
	sectionSuffixPattern = regexp.MustCompile(`(?i)^(.*\S)[,\s]+((?:lecture|lec|laboratory|lab|tutorial|tut|practical|prac|seminar|sem|workshop|wshop|discussion|dis|test|tes|exam|eax|optional|opt)(?:[ -]*[A-Z0-9]+)?)$`)
	testEventPattern     = regexp.MustCompile(`(?i)(^|[^a-z0-9])(?:tests?|tes(?:[ -]?[a-z0-9]{1,2})?|exams?(?:[ -]?[a-z0-9]{1,2})?|eax(?:[ -]?[a-z0-9]{1,2})?)(?:$|[^a-z0-9])`)
)

// buildEventGroups groups related calendar summaries while keeping every
// original summary as the value submitted for filtering.
func buildEventGroups(summaries []string, selected map[string]bool) []eventGroup {
	builders := make(map[string]*eventGroupBuilder)
	keys := make([]string, 0)
	ungrouped := make([]eventOption, 0)

	for _, summary := range summaries {
		isTest := testEventPattern.MatchString(summary)
		name, label, ok := eventGroupParts(summary)
		if !ok {
			ungrouped = append(ungrouped, eventOption{Summary: summary, Label: summary, Selected: selected[summary], IsTest: isTest})
			continue
		}
		key := strings.ToLower(name)
		builder, exists := builders[key]
		if !exists {
			builder = &eventGroupBuilder{name: name}
			builders[key] = builder
			keys = append(keys, key)
		}
		builder.options = append(builder.options, eventOption{Summary: summary, Label: label, Selected: selected[summary], IsTest: isTest})
	}

	groups := make([]eventGroup, 0, len(keys))
	for _, key := range keys {
		builder := builders[key]
		if len(builder.options) < 2 {
			for _, option := range builder.options {
				ungrouped = append(ungrouped, eventOption{Summary: option.Summary, Label: option.Summary, Selected: selected[option.Summary], IsTest: option.IsTest})
			}
			continue
		}
		group := eventGroup{Name: builder.name, Options: builder.options}
		updateEventGroupState(&group)
		groups = append(groups, group)
	}
	if len(ungrouped) > 0 {
		other := eventGroup{Name: "Other event types", Options: ungrouped}
		updateEventGroupState(&other)
		groups = append(groups, other)
	}
	for index := range groups {
		groups[index].ID = strconv.Itoa(index)
	}
	return groups
}

func updateEventGroupState(group *eventGroup) {
	group.AllSelected = len(group.Options) > 0
	group.SelectedTestTypes = nil
	for _, option := range group.Options {
		if !option.Selected {
			group.AllSelected = false
		}
		if option.IsTest && option.Selected {
			group.SelectedTestTypes = append(group.SelectedTestTypes, option.Summary)
		}
	}
}

func eventGroupParts(summary string) (name string, label string, ok bool) {
	if parts := colonCoursePattern.FindStringSubmatch(summary); len(parts) == 3 {
		return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), true
	}
	if parts := numberCoursePattern.FindStringSubmatch(summary); len(parts) == 3 {
		return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), true
	}
	if parts := sectionSuffixPattern.FindStringSubmatch(summary); len(parts) == 3 {
		return strings.TrimRight(strings.TrimSpace(parts[1]), ",:-"), strings.TrimSpace(parts[2]), true
	}
	return "", "", false
}
