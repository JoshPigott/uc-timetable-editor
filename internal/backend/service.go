package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxCalendarBytes = 10 << 20

// Service coordinates feed storage and calendar fetching.
type Service struct {
	repository FeedRepository
	client     *http.Client
}

// NewService creates the backend use cases with a bounded calendar client.
func NewService(repository FeedRepository) *Service {
	return &Service{
		repository: repository,
		client: &http.Client{
			Timeout: 25 * time.Second,
			CheckRedirect: func(request *http.Request, previous []*http.Request) error {
				if len(previous) >= 5 || !AllowedCalendarURL(request.URL) {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// NewServiceWithClient uses a supplied transport while keeping the feed redirect policy.
func NewServiceWithClient(repository FeedRepository, client *http.Client) *Service {
	service := NewService(repository)
	if client == nil {
		return service
	}
	copy := *client
	if copy.Timeout <= 0 || copy.Timeout > 25*time.Second {
		copy.Timeout = 25 * time.Second
	}
	copy.CheckRedirect = service.client.CheckRedirect
	service.client = &copy
	return service
}

// CreateFeed validates and saves a source URL and its event filters.
func (s *Service) CreateFeed(ctx context.Context, sourceURL string, rules Filters) (string, error) {
	parsed, err := url.Parse(sourceURL)
	if err != nil || !AllowedCalendarURL(parsed) {
		return "", errors.New("enter a supported HTTPS calendar iCal URL")
	}
	cleanRules, err := normalizeFilters(rules)
	if err != nil {
		return "", err
	}
	return s.repository.Create(ctx, FeedConfig{SourceURL: sourceURL, Filters: cleanRules})
}

// EventTypes fetches a source calendar and returns its distinct event titles.
func (s *Service) EventTypes(ctx context.Context, source string) ([]string, error) {
	body, err := s.fetchCalendar(ctx, source)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -7)
	return CalendarEventTypesSince(body, since)
}

// Calendar loads a saved feed, applies its selected event types, and returns iCal data.
func (s *Service) Calendar(ctx context.Context, token string) ([]byte, error) {
	config, err := s.repository.Lookup(ctx, token)
	if err != nil {
		return nil, err
	}
	body, err := s.fetchCalendar(ctx, config.SourceURL)
	if err != nil {
		return nil, err
	}
	return FilterCalendar(body, config.Filters)
}

func (s *Service) fetchCalendar(ctx context.Context, source string) ([]byte, error) {
	sourceURL, err := url.Parse(source)
	if err != nil || !AllowedCalendarURL(sourceURL) {
		return nil, errors.New("enter a supported HTTPS calendar iCal URL")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("build calendar request: %w", err)
	}
	request.Header.Set("Accept", "text/calendar, text/plain;q=0.9, */*;q=0.1")
	response, err := s.client.Do(request)
	if err != nil {
		var requestErr *url.Error
		if errors.As(err, &requestErr) {
			// Log only the host and underlying transport error; the calendar URL contains a private token.
			log.Printf("calendar request failed for host %s: %v", sourceURL.Hostname(), requestErr.Err)
		} else {
			log.Printf("calendar request failed for host %s (error type %T)", sourceURL.Hostname(), err)
		}
		return nil, errors.New("calendar source unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		log.Printf("calendar request to host %s returned HTTP status %d", sourceURL.Hostname(), response.StatusCode)
		return nil, errors.New("calendar source unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxCalendarBytes+1))
	if err != nil || len(body) > maxCalendarBytes {
		return nil, errors.New("calendar source returned an invalid or oversized calendar")
	}
	return body, nil
}

// AllowedCalendarURL limits feed fetching to supported HTTPS iCal hosts.
func AllowedCalendarURL(source *url.URL) bool {
	if source == nil || source.Scheme != "https" || source.User != nil || source.Fragment != "" {
		return false
	}
	host := strings.ToLower(source.Hostname())
	if host != "timetable.canterbury.ac.nz" {
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
	if len(terms) > 40 {
		return Filters{}, errors.New("add no more than 40 filter phrases")
	}
	excludedSummaryTypes := make([]string, 0, len(rules.ExcludedSummaryTypes))
	seenSummaries := make(map[string]bool)
	for _, summary := range rules.ExcludedSummaryTypes {
		summary = strings.TrimSpace(summary)
		if summary == "" {
			continue
		}
		key := strings.ToLower(summary)
		if !seenSummaries[key] {
			excludedSummaryTypes = append(excludedSummaryTypes, summary)
			seenSummaries[key] = true
		}
	}
	return Filters{Terms: terms, Fields: fields, ExcludedSummaryTypes: excludedSummaryTypes, FilterSummary: rules.FilterSummary}, nil
}
