package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"timetable-editor/internal/backend"
	"timetable-editor/internal/database"
	"timetable-editor/internal/handler"
	"timetable-editor/internal/router"
)

const sourceURL = "https://timetable.canterbury.ac.nz/ical/course.ics"

const calendarData = "BEGIN:VCALENDAR\n" +
	"VERSION:2.0\n" +
	"BEGIN:VEVENT\nUID:lecture\nDTSTART:20990101T100000Z\nSUMMARY:Lecture\nEND:VEVENT\n" +
	"BEGIN:VEVENT\nUID:lab\nDTSTART:20990101T110000Z\nSUMMARY:Lab\nEND:VEVENT\n" +
	"END:VCALENDAR\n"

type staticCalendarTransport struct{}

func (staticCalendarTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Method != http.MethodGet || request.URL.String() != sourceURL {
		return nil, fmt.Errorf("unexpected calendar request: %s %s", request.Method, request.URL)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/calendar"}},
		Body:       io.NopCloser(strings.NewReader(calendarData)),
		Request:    request,
	}, nil
}

type tokenCapturingRepository struct {
	*database.Store
	token string
}

func (r *tokenCapturingRepository) Create(ctx context.Context, config backend.FeedConfig) (string, error) {
	token, err := r.Store.Create(ctx, config)
	if err == nil {
		r.token = token
	}
	return token, err
}

func TestCreateFeedThenRetrieveFilteredCalendar(t *testing.T) {
	server, repository := newTestServer(t)
	client := server.Client()

	analyzeStatus, analyzeBody := postForm(t, client, server.URL+"/analyze", url.Values{
		"feed_url": {sourceURL},
	})
	if analyzeStatus != http.StatusOK {
		t.Fatalf("analyze status = %d, want %d: %s", analyzeStatus, http.StatusOK, analyzeBody)
	}
	for _, eventType := range []string{"Lecture", "Lab"} {
		if !strings.Contains(analyzeBody, eventType) {
			t.Fatalf("analyze response is missing event type %q: %s", eventType, analyzeBody)
		}
	}

	createStatus, createBody := postForm(t, client, server.URL+"/create", url.Values{
		"feed_url":          {sourceURL},
		"event_types_ready": {"true"},
		"filter_summary":    {"true"},
		"available_summary": {"Lecture", "Lab"},
		"summary_type":      {"Lecture"},
	})
	if createStatus != http.StatusOK {
		t.Fatalf("create status = %d, want %d: %s", createStatus, http.StatusOK, createBody)
	}
	if repository.token == "" {
		t.Fatal("create did not save a feed")
	}
	feedURL := "https://calendar.example.test/feed/" + repository.token + "/calendar.ics"
	if !strings.Contains(createBody, feedURL) {
		t.Fatalf("create response is missing feed URL %q", feedURL)
	}

	feedStatus, feedBody := get(t, client, server.URL+"/feed/"+repository.token+"/calendar.ics")
	if feedStatus != http.StatusOK {
		t.Fatalf("feed status = %d, want %d: %s", feedStatus, http.StatusOK, feedBody)
	}
	if !strings.Contains(feedBody, "SUMMARY:Lecture\r\n") {
		t.Errorf("feed omitted selected event Lecture: %s", feedBody)
	}
	if strings.Contains(feedBody, "SUMMARY:Lab\r\n") {
		t.Errorf("feed included deselected event Lab: %s", feedBody)
	}
}

func newTestServer(t *testing.T) (*httptest.Server, *tokenCapturingRepository) {
	t.Helper()

	store, err := database.Open(filepath.Join(t.TempDir(), "feeds.db"), backend.NewSecretBox([]byte("integration test master key")))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	repository := &tokenCapturingRepository{Store: store}
	service := backend.NewServiceWithClient(repository, &http.Client{Transport: staticCalendarTransport{}})
	app, err := handler.New(service, "https://calendar.example.test")
	if err != nil {
		t.Fatalf("create handler: %v", err)
	}
	server := httptest.NewServer(router.New(app))
	t.Cleanup(server.Close)
	return server, repository
}

func postForm(t *testing.T, client *http.Client, target string, values url.Values) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, target, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatalf("create POST request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	return doRequest(t, client, request)
}

func get(t *testing.T, client *http.Client, target string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("create GET request: %v", err)
	}
	return doRequest(t, client, request)
}

func doRequest(t *testing.T, client *http.Client, request *http.Request) (int, string) {
	t.Helper()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send %s request: %v", request.Method, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", request.Method, err)
	}
	return response.StatusCode, string(body)
}
