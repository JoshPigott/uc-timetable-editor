package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"timetable-editor/internal/backend"

	"github.com/mattn/go-sqlite3"
)

// Store persists encrypted feed configurations in SQLite.
type Store struct {
	db  *sql.DB
	box *backend.SecretBox
}

// Open creates the database directory, applies safe SQLite settings, and prepares the feed table.
func Open(path string, box *backend.SecretBox) (*Store, error) {
	if path != ":memory:" {
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve database path: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0700); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to SQLite database: %w", err)
	}
	for _, pragma := range []string{"PRAGMA journal_mode=DELETE", "PRAGMA secure_delete=ON", "PRAGMA busy_timeout=5000"} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
	}
	if path != ":memory:" {
		if err := os.Chmod(path, 0600); err != nil && !errors.Is(err, os.ErrPermission) {
			_ = db.Close()
			return nil, fmt.Errorf("set database file permissions: %w", err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS feeds (
		id_hash TEXT PRIMARY KEY,
		url_hash TEXT NOT NULL,
		payload BLOB NOT NULL,
		created_at INTEGER NOT NULL
	)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create feeds table: %w", err)
	}
	return &Store{db: db, box: box}, nil
}

// Close releases SQLite resources held by the store.
func (s *Store) Close() error {
	return s.db.Close()
}

// Create encrypts a feed config and saves only keyed digests and ciphertext.
func (s *Store) Create(ctx context.Context, config backend.FeedConfig) (string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		token, err := randomToken()
		if err != nil {
			return "", err
		}
		rowID := s.box.FeedTokenDigest(token)
		plaintext, err := json.Marshal(config)
		if err != nil {
			return "", fmt.Errorf("encode feed config: %w", err)
		}
		payload, err := s.box.Seal(rowID, plaintext)
		if err != nil {
			return "", fmt.Errorf("encrypt feed config: %w", err)
		}
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO feeds(id_hash, url_hash, payload, created_at) VALUES(?, ?, ?, ?)`,
			rowID, s.box.SourceURLDigest(config.SourceURL), payload, time.Now().Unix())
		if err == nil {
			return token, nil
		}
		var sqliteErr sqlite3.Error
		if !errors.As(err, &sqliteErr) || sqliteErr.Code != sqlite3.ErrConstraint {
			return "", fmt.Errorf("save feed config: %w", err)
		}
	}
	return "", errors.New("could not allocate a unique feed token")
}

// Lookup finds a feed by token digest and decrypts its configuration.
func (s *Store) Lookup(ctx context.Context, token string) (backend.FeedConfig, error) {
	rowID := s.box.FeedTokenDigest(token)
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM feeds WHERE id_hash = ?`, rowID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return backend.FeedConfig{}, backend.ErrFeedNotFound
	}
	if err != nil {
		return backend.FeedConfig{}, fmt.Errorf("load feed config: %w", err)
	}
	plaintext, err := s.box.Open(rowID, payload)
	if err != nil {
		return backend.FeedConfig{}, fmt.Errorf("decrypt feed config: %w", err)
	}
	var config backend.FeedConfig
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return backend.FeedConfig{}, fmt.Errorf("decode feed config: %w", err)
	}
	return config, nil
}

// Update encrypts a replacement configuration for an existing feed token.
func (s *Store) Update(ctx context.Context, token string, config backend.FeedConfig) error {
	rowID := s.box.FeedTokenDigest(token)
	plaintext, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode feed config: %w", err)
	}
	payload, err := s.box.Seal(rowID, plaintext)
	if err != nil {
		return fmt.Errorf("encrypt feed config: %w", err)
	}
	result, err := s.db.ExecContext(ctx,
		`UPDATE feeds SET url_hash = ?, payload = ? WHERE id_hash = ?`,
		s.box.SourceURLDigest(config.SourceURL), payload, rowID)
	if err != nil {
		return fmt.Errorf("update feed config: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check feed update: %w", err)
	}
	if rows == 0 {
		return backend.ErrFeedNotFound
	}
	return nil
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate feed token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
