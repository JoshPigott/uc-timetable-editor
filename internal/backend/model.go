package backend

import (
	"context"
	"errors"
)

// ErrFeedNotFound identifies an unknown public feed token.
var ErrFeedNotFound = errors.New("feed not found")

// Filters describes phrases that should remove matching calendar events.
type Filters struct {
	Terms  []string `json:"terms"`
	Fields []string `json:"fields"`
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
