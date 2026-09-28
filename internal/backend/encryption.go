package backend

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

// SecretBox separates encryption and lookup keys derived from one master key.
type SecretBox struct {
	encryptionKey []byte
	lookupKey     []byte
	urlHashKey    []byte
}

// NewSecretBox derives independent keys for encrypted records and HMAC digests.
func NewSecretBox(masterKey []byte) *SecretBox {
	return &SecretBox{
		encryptionKey: deriveKey(masterKey, "feed-payload-encryption-v1"),
		lookupKey:     deriveKey(masterKey, "public-feed-lookup-v1"),
		urlHashKey:    deriveKey(masterKey, "source-url-hash-v1"),
	}
}

// FeedTokenDigest returns the database lookup digest for a public feed token.
func (s *SecretBox) FeedTokenDigest(token string) string {
	return hmacDigest(s.lookupKey, token)
}

// SourceURLDigest returns a keyed, one-way digest of a source calendar URL.
func (s *SecretBox) SourceURLDigest(sourceURL string) string {
	return hmacDigest(s.urlHashKey, sourceURL)
}

// Seal encrypts plaintext and authenticates it against its row identifier.
func (s *SecretBox) Seal(rowID string, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	stored := append([]byte(nil), nonce...)
	return gcm.Seal(stored, nonce, plaintext, []byte(rowID)), nil
}

// Open verifies and decrypts a record encrypted by Seal.
func (s *SecretBox) Open(rowID string, payload []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize()+gcm.Overhead() {
		return nil, errors.New("encrypted feed payload is too short")
	}
	return gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], []byte(rowID))
}

func deriveKey(masterKey []byte, purpose string) []byte {
	mac := hmac.New(sha256.New, masterKey)
	_, _ = io.WriteString(mac, "timetable-filter/"+purpose)
	return mac.Sum(nil)
}

func hmacDigest(key []byte, value string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = io.WriteString(mac, value)
	return hex.EncodeToString(mac.Sum(nil))
}
