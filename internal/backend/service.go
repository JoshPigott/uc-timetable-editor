package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxCalendarBytes = 10 << 20

// Service coordinates feed storage, Google Calendar fetching, and iCal filtering.
type Service struct {
	repository FeedRepository
	client     *http.Client
}

// NewService creates the backend use cases with a bounded Google feed client.
func NewService(repository FeedRepository) *Service {
	return &Service{
		repository: repository,
		client: &http.Client{
			Timeout: 25 * time.Second,
			CheckRedirect: func(request *http.Request, previous []*http.Request) error {
				if len(previous) >= 5 || !AllowedGoogleCalendarURL(request.URL) {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// CreateFeed validates and saves a source URL and its event filters.
func (s *Service) CreateFeed(ctx context.Context, sourceURL string, rules Filters) (string, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil || !AllowedGoogleCalendarURL(parsed) {
		return "", errors.New("enter a Google Calendar iCal URL using HTTPS")
	}
	cleanRules, err := normalizeFilters(rules)
	if err != nil {
		return "", err
	}
	return s.repository.Create(ctx, FeedConfig{SourceURL: sourceURL, Filters: cleanRules})
}

// FilteredCalendar loads a saved feed, fetches its source, and removes matching events.
func (s *Service) FilteredCalendar(ctx context.Context, token string) ([]byte, error) {
	config, err := s.repository.Lookup(ctx, token)
	if err != nil {
		return nil, err
	}
	sourceURL, err := url.Parse(config.SourceURL)
	if err != nil || !AllowedGoogleCalendarURL(sourceURL) {
		return nil, errors.New("saved calendar source is invalid")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, config.SourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build calendar request: %w", err)
	}
	request.Header.Set("Accept", "text/calendar, text/plain;q=0.9, */*;q=0.1")
	response, err := s.client.Do(request)
	if err != nil {
		return nil, errors.New("calendar source unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, errors.New("calendar source unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxCalendarBytes+1))
	if err != nil || len(body) > maxCalendarBytes {
		return nil, errors.New("calendar source returned an invalid or oversized calendar")
	}
	return FilterCalendar(body, config.Filters)
}

// AllowedGoogleCalendarURL limits feed fetching to HTTPS Google Calendar iCal URLs.
func AllowedGoogleCalendarURL(source *url.URL) bool {
	if source == nil || source.Scheme != "https" || !strings.EqualFold(source.Hostname(), "calendar.google.com") || source.User != nil || source.Fragment != "" {
		return false
	}
	if port := source.Port(); port != "" && port != "443" {
		return false
	}
	return strings.Contains(strings.ToLower(source.EscapedPath()), "/ical/")
}

func normalizeFilters(rules Filters) (Filters, error) {
	validFields := map[string]bool{"summary": true, "description": true, "location": true}
	fields := make([]string, 0, len(rules.Fields))
	seenFields := make(map[string]bool)
	for _, field := range rules.Fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if validFields[field] && !seenFields[field] {
			fields = append(fields, field)
			seenFields[field] = true
		}
	}
	if len(fields) == 0 {
		return Filters{}, errors.New("select at least one event field to search")
	}
	terms := make([]string, 0, len(rules.Terms))
	seenTerms := make(map[string]bool)
	for _, term := range rules.Terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		if len([]rune(term)) > 160 {
			return Filters{}, errors.New("each filter phrase must be 160 characters or fewer")
		}
		key := strings.ToLower(term)
		if !seenTerms[key] {
			terms = append(terms, term)
			seenTerms[key] = true
		}
	}
	if len(terms) == 0 || len(terms) > 40 {
		return Filters{}, errors.New("add between 1 and 40 filter phrases")
	}
	return Filters{Terms: terms, Fields: fields}, nil
}
