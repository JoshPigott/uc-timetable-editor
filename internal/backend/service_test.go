package backend

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestAllowedCalendarURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "supported calendar", url: "https://timetable.canterbury.ac.nz/ical/course.ics", want: true},
		{name: "default HTTPS port", url: "https://timetable.canterbury.ac.nz:443/ical/course.ics", want: true},
		{name: "HTTP is rejected", url: "http://timetable.canterbury.ac.nz/ical/course.ics"},
		{name: "other host is rejected", url: "https://example.com/ical/course.ics"},
		{name: "host suffix is rejected", url: "https://timetable.canterbury.ac.nz.example.com/ical/course.ics"},
		{name: "credentials are rejected", url: "https://user@timetable.canterbury.ac.nz/ical/course.ics"},
		{name: "nondefault port is rejected", url: "https://timetable.canterbury.ac.nz:8443/ical/course.ics"},
		{name: "non iCal path is rejected", url: "https://timetable.canterbury.ac.nz/calendar/course.ics"},
		{name: "fragment is rejected", url: "https://timetable.canterbury.ac.nz/ical/course.ics#private"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := url.Parse(test.url)
			if err != nil {
				t.Fatalf("url.Parse() error = %v", err)
			}
			if got := AllowedCalendarURL(parsed); got != test.want {
				t.Errorf("AllowedCalendarURL(%q) = %v, want %v", test.url, got, test.want)
			}
		})
	}

	if AllowedCalendarURL(nil) {
		t.Error("AllowedCalendarURL(nil) = true, want false")
	}
}

func TestNormalizeFiltersTrimsAndDeduplicates(t *testing.T) {
	got, err := normalizeFilters(Filters{
		Fields:               []string{" Summary ", "summary", "Description", "invalid"},
		Terms:                []string{" lab ", "LAB", ""},
		ExcludedSummaryTypes: []string{" Lecture ", "lecture", "Seminar"},
		FilterSummary:        true,
	})
	if err != nil {
		t.Fatalf("normalizeFilters() error = %v", err)
	}
	want := Filters{
		Fields:               []string{"summary", "description"},
		Terms:                []string{"lab"},
		ExcludedSummaryTypes: []string{"Lecture", "Seminar"},
		FilterSummary:        true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeFilters() = %#v, want %#v", got, want)
	}
}

func TestNormalizeFiltersEnforcesPhraseLimits(t *testing.T) {
	fortyTerms := make([]string, 40)
	fortyOneTerms := make([]string, 41)
	for i := range fortyOneTerms {
		fortyOneTerms[i] = fmt.Sprintf("phrase %d", i)
		if i < len(fortyTerms) {
			fortyTerms[i] = fortyOneTerms[i]
		}
	}

	tests := []struct {
		name    string
		rules   Filters
		wantErr bool
	}{
		{name: "160 rune phrase is allowed", rules: Filters{Terms: []string{strings.Repeat("界", 160)}}},
		{name: "phrase over 160 runes is rejected", rules: Filters{Terms: []string{strings.Repeat("界", 161)}}, wantErr: true},
		{name: "40 phrases are allowed", rules: Filters{Terms: fortyTerms}},
		{name: "more than 40 phrases are rejected", rules: Filters{Terms: fortyOneTerms}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeFilters(test.rules)
			if (err != nil) != test.wantErr {
				t.Errorf("normalizeFilters() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
