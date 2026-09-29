package backend

import (
	"context"
	"errors"
)

// ErrFeedNotFound identifies an unknown public feed token.
var ErrFeedNotFound = errors.New("feed not found")

// Filters stores the event types selected for a calendar feed.
type Filters struct {
	Terms                []string `json:"terms,omitempty"`
	Fields               []string `json:"fields,omitempty"`
	ExcludedSummaryTypes []string `json:"excluded_summary_types,omitempty"`
	FilterSummary        bool     `json:"filter_summary,omitempty"`
}

// FeedConfig is the sensitive source URL and its filtering rules.
type FeedConfig struct {
	SourceURL string  `json:"source_url"`
	Filters   Filters `json:"filters"`
}

// FeedRepository stores feed configs without exposing their clear text to SQLite.
type FeedRepository interface {
	Create(context.Context, FeedConfig) (string, error)
	Lookup(context.Context, string) (FeedConfig, error)
}
