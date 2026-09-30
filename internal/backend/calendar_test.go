package backend

import (
	"reflect"
	"testing"
)

func TestCalendarEventTypesDeduplicatesAndUnescapesSummaries(t *testing.T) {
	data := "BEGIN:VCALENDAR\r\n" +
		"SUMMARY:Outside event\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY;LANGUAGE=en:Lab\\, Studio\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:lecture\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"SUMMARY:LAB\\, STUDIO\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	got, err := CalendarEventTypes([]byte(data))
	if err != nil {
		t.Fatalf("CalendarEventTypes() error = %v", err)
	}
	want := []string{"Lab, Studio", "lecture"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CalendarEventTypes() = %#v, want %#v", got, want)
	}
}

func TestCalendarEventTypesRejectsMalformedEvents(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "nested event",
			data: "BEGIN:VCALENDAR\nBEGIN:VEVENT\nBEGIN:VEVENT\nEND:VEVENT\nEND:VCALENDAR\n",
		},
		{
			name: "unmatched event end",
			data: "BEGIN:VCALENDAR\nEND:VEVENT\nEND:VCALENDAR\n",
		},
		{
			name: "incomplete calendar",
			data: "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Lab\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CalendarEventTypes([]byte(test.data)); err == nil {
				t.Fatal("CalendarEventTypes() error = nil, want malformed calendar error")
			}
		})
	}
}

func TestFilterCalendarExcludesEventsMatchingTerms(t *testing.T) {
	data := "BEGIN:VCALENDAR\r\n" +
		"X-WR-CALNAME:Schedule\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:lab\r\n" +
		"SUMMARY:Lab session\r\n" +
		"END:VEVENT\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:lecture\r\n" +
		"SUMMARY:Lecture\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	got, err := FilterCalendar([]byte(data), Filters{Terms: []string{"LAB"}, Fields: []string{"summary"}})
	if err != nil {
		t.Fatalf("FilterCalendar() error = %v", err)
	}
	want := "BEGIN:VCALENDAR\r\n" +
		"X-WR-CALNAME:Schedule\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:lecture\r\n" +
		"SUMMARY:Lecture\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"
	if string(got) != want {
		t.Errorf("FilterCalendar() = %q, want %q", got, want)
	}
}

func TestFilterCalendarExcludesSelectedSummaryTypes(t *testing.T) {
	data := "BEGIN:VCALENDAR\n" +
		"BEGIN:VEVENT\nSUMMARY:Lecture\nEND:VEVENT\n" +
		"BEGIN:VEVENT\nSUMMARY:Seminar\nEND:VEVENT\n" +
		"END:VCALENDAR\n"

	got, err := FilterCalendar([]byte(data), Filters{
		FilterSummary:        true,
		ExcludedSummaryTypes: []string{" seminar "},
	})
	if err != nil {
		t.Fatalf("FilterCalendar() error = %v", err)
	}
	want := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:Lecture\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"
	if string(got) != want {
		t.Errorf("FilterCalendar() = %q, want %q", got, want)
	}
}

func TestFilterCalendarKeepsAllEventsWhenNoSummaryTypesAreExcluded(t *testing.T) {
	data := "BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:Lab\nEND:VEVENT\nEND:VCALENDAR\n"
	got, err := FilterCalendar([]byte(data), Filters{FilterSummary: true})
	if err != nil {
		t.Fatalf("FilterCalendar() error = %v", err)
	}
	want := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:Lab\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	if string(got) != want {
		t.Errorf("FilterCalendar() = %q, want %q", got, want)
	}
}
