package backend

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// CalendarEventTypes returns the distinct SUMMARY values in an iCalendar feed.
func CalendarEventTypes(data []byte) ([]string, error) {
	return calendarEventTypes(data, nil)
}

// CalendarEventTypesSince returns distinct SUMMARY values for events that
// start on or after the supplied time.
func CalendarEventTypesSince(data []byte, since time.Time) ([]string, error) {
	return calendarEventTypes(data, &since)
}

func calendarEventTypes(data []byte, since *time.Time) ([]string, error) {
	lines := calendarLines(data)
	titles := make([]string, 0)
	seen := make(map[string]bool)
	inEvent := false
	calendarStarted, calendarEnded := false, false
	eventTitles := make([]string, 0, 1)
	var eventStart time.Time
	hasEventStart := false
	for _, line := range lines {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if !inEvent && upper == "BEGIN:VCALENDAR" {
			calendarStarted = true
		}
		if !inEvent && upper == "END:VCALENDAR" {
			calendarEnded = true
		}
		if upper == "BEGIN:VEVENT" {
			if inEvent {
				return nil, errors.New("calendar contains a nested VEVENT")
			}
			inEvent = true
			eventTitles = eventTitles[:0]
			eventStart = time.Time{}
			hasEventStart = false
			continue
		}
		if upper == "END:VEVENT" {
			if !inEvent {
				return nil, errors.New("calendar contains an unmatched VEVENT")
			}
			inEvent = false
			if since == nil || (hasEventStart && !eventStart.Before(*since)) {
				for _, eventTitle := range eventTitles {
					key := strings.ToLower(eventTitle)
					if !seen[key] {
						titles = append(titles, eventTitle)
						seen[key] = true
					}
				}
			}
			continue
		}
		if !inEvent {
			continue
		}
		if !hasEventStart {
			if start, ok := eventStartValue(line); ok {
				eventStart, hasEventStart = start, true
			}
		}
		title, ok := summaryValue(line)
		if !ok || title == "" {
			continue
		}
		eventTitles = append(eventTitles, title)
	}
	if inEvent || !calendarStarted || !calendarEnded {
		return nil, errors.New("calendar is incomplete")
	}
	sort.SliceStable(titles, func(i, j int) bool {
		return strings.ToLower(titles[i]) < strings.ToLower(titles[j])
	})
	return titles, nil
}

func eventStartValue(line string) (time.Time, bool) {
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return time.Time{}, false
	}
	parts := strings.Split(line[:colon], ";")
	if len(parts) == 0 || !strings.EqualFold(strings.TrimSpace(parts[0]), "DTSTART") {
		return time.Time{}, false
	}

	location := time.Local
	for _, parameter := range parts[1:] {
		name, value, ok := strings.Cut(parameter, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "TZID") {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if zone, err := time.LoadLocation(value); err == nil {
			location = zone
		}
		break
	}

	value := strings.TrimSpace(line[colon+1:])
	if strings.HasSuffix(value, "Z") {
		for _, layout := range []string{"20060102T150405Z", "20060102T1504Z"} {
			if parsed, err := time.Parse(layout, value); err == nil {
				return parsed, true
			}
		}
		return time.Time{}, false
	}
	for _, layout := range []string{"20060102T150405", "20060102T1504", "20060102"} {
		if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

// FilterCalendar keeps VEVENT blocks that match the configured event filters.
func FilterCalendar(data []byte, rules Filters) ([]byte, error) {
	lines := calendarLines(data)

	output := make([]string, 0, len(lines))
	event := make([]string, 0, 16)
	inEvent := false
	calendarStarted, calendarEnded := false, false
	for _, line := range lines {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if !inEvent && upper == "BEGIN:VCALENDAR" {
			calendarStarted = true
		}
		if !inEvent && upper == "END:VCALENDAR" {
			calendarEnded = true
		}
		if upper == "BEGIN:VEVENT" {
			if inEvent {
				return nil, errors.New("calendar contains a nested VEVENT")
			}
			inEvent = true
			event = append(event[:0], line)
			continue
		}
		if inEvent {
			event = append(event, line)
			if upper == "END:VEVENT" {
				if !eventMatches(event, rules) {
					output = append(output, event...)
				}
				event = event[:0]
				inEvent = false
			}
			continue
		}
		output = append(output, line)
	}
	if inEvent || !calendarStarted || !calendarEnded {
		return nil, errors.New("calendar is incomplete")
	}
	var result strings.Builder
	for _, line := range output {
		writeFoldedLine(&result, line)
	}
	return []byte(result.String()), nil
}

func calendarLines(data []byte) []string {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	physical := strings.Split(text, "\n")
	if len(physical) > 0 && physical[len(physical)-1] == "" {
		physical = physical[:len(physical)-1]
	}
	return unfoldLines(physical)
}

func unfoldLines(physical []string) []string {
	lines := make([]string, 0, len(physical))
	for _, line := range physical {
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func eventMatches(lines []string, rules Filters) bool {
	if rules.FilterSummary {
		excluded := make(map[string]bool, len(rules.ExcludedSummaryTypes))
		for _, summary := range rules.ExcludedSummaryTypes {
			excluded[strings.ToLower(strings.TrimSpace(summary))] = true
		}
		for _, line := range lines {
			if summary, ok := summaryValue(line); ok && excluded[strings.ToLower(summary)] {
				return true
			}
		}
		return false
	}
	if len(rules.Terms) == 0 {
		return true
	}
	wanted := make(map[string]bool, len(rules.Fields))
	for _, field := range rules.Fields {
		wanted[strings.ToLower(field)] = true
	}
	fields := map[string]*strings.Builder{
		"summary": new(strings.Builder), "description": new(strings.Builder), "location": new(strings.Builder),
	}
	for _, line := range lines {
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		name := strings.ToUpper(strings.SplitN(line[:colon], ";", 2)[0])
		field := strings.ToLower(name)
		if builder, ok := fields[field]; ok {
			builder.WriteString(unescapeText(line[colon+1:]))
		}
	}
	for field, builder := range fields {
		if !wanted[field] {
			continue
		}
		value := strings.ToLower(builder.String())
		for _, term := range rules.Terms {
			if strings.Contains(value, strings.ToLower(term)) {
				return true
			}
		}
	}
	return false
}

func summaryValue(line string) (string, bool) {
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return "", false
	}
	name := strings.ToUpper(strings.SplitN(line[:colon], ";", 2)[0])
	if name != "SUMMARY" {
		return "", false
	}
	return strings.TrimSpace(unescapeText(line[colon+1:])), true
}

func unescapeText(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' || index+1 >= len(value) {
			result.WriteByte(value[index])
			continue
		}
		index++
		switch value[index] {
		case 'n', 'N':
			result.WriteByte('\n')
		case ',', ';', '\\':
			result.WriteByte(value[index])
		default:
			result.WriteByte('\\')
			result.WriteByte(value[index])
		}
	}
	return result.String()
}

func writeFoldedLine(output *strings.Builder, line string) {
	const maxOctets = 75
	remaining := line
	firstLine := true
	for len(remaining) > 0 {
		limit := maxOctets
		if !firstLine {
			output.WriteByte(' ')
			limit--
		}
		used, cut := 0, 0
		for cut < len(remaining) {
			_, size := utf8.DecodeRuneInString(remaining[cut:])
			if used+size > limit {
				break
			}
			used += size
			cut += size
		}
		if cut == 0 {
			_, cut = utf8.DecodeRuneInString(remaining)
		}
		output.WriteString(remaining[:cut])
		output.WriteString("\r\n")
		remaining = remaining[cut:]
		firstLine = false
	}
	if firstLine {
		output.WriteString("\r\n")
	}
}
