package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"

	"timetable-editor/internal/backend"
)

// Config contains runtime settings for the local server.
type Config struct {
	Address       string
	DatabasePath  string
	PublicBaseURL string
	MasterKey     []byte
	OpenBrowser   bool
}

// Load reads settings from the environment and creates the local key on first use.
func Load() (Config, error) {
	masterKey, err := backend.LoadMasterKey(os.Getenv("TIMETABLE_MASTER_KEY"), os.Getenv("TIMETABLE_KEY_FILE"))
	if err != nil {
		return Config{}, err
	}
	address := os.Getenv("ADDR")
	if address == "" {
		if port := os.Getenv("PORT"); port != "" {
			address = net.JoinHostPort("0.0.0.0", port)
		} else {
			address = "127.0.0.1:8080"
		}
	}
	databasePath := os.Getenv("TIMETABLE_DB_PATH")
	if databasePath == "" {
		databasePath = "timetable.db"
	}
	baseURL := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return Config{}, errors.New("PUBLIC_BASE_URL must be an HTTP(S) origin without credentials, path, query, or fragment")
		}
	}

	return Config{
		Address: address, DatabasePath: databasePath, PublicBaseURL: baseURL,
		MasterKey: masterKey, OpenBrowser: envBool("OPEN_BROWSER", true),
	}, nil
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
