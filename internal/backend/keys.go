package backend

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LoadMasterKey uses the configured key or creates one durable per-user key.
func LoadMasterKey(environmentValue, keyFile string) ([]byte, error) {
	if environmentValue != "" {
		return decodeMasterKey(environmentValue)
	}
	if keyFile == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("find user config directory for encryption key: %w", err)
		}
		keyFile = filepath.Join(configDir, "timetable-filter", "master.key")
	}
	return loadOrCreateKey(keyFile)
}

func decodeMasterKey(value string) ([]byte, error) {
	if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		decoded, err := encoding.DecodeString(value)
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}
	return nil, errors.New("TIMETABLE_MASTER_KEY must be a 32-byte base64 value or 64 hexadecimal characters")
}

func loadOrCreateKey(path string) ([]byte, error) {
	if key, err := readKey(path); err == nil {
		return key, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create encryption key directory: %w", err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate encryption key: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		return readKey(path)
	}
	if err != nil {
		return nil, fmt.Errorf("create encryption key file: %w", err)
	}
	if _, err := file.Write(key); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("write encryption key file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("sync encryption key file: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close encryption key file: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil && !errors.Is(err, os.ErrPermission) {
		return nil, fmt.Errorf("set encryption key file permissions: %w", err)
	}
	return key, nil
}

func readKey(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	key, err := io.ReadAll(io.LimitReader(file, 33))
	if err != nil {
		return nil, fmt.Errorf("read encryption key file: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("encryption key file must contain exactly 32 bytes")
	}
	if err := os.Chmod(path, 0600); err != nil && !errors.Is(err, os.ErrPermission) {
		return nil, fmt.Errorf("set encryption key file permissions: %w", err)
	}
	return key, nil
}
