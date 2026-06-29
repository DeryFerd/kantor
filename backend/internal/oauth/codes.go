package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"sync"
	"time"
)

const codeTTL = 2 * time.Minute

type AuthCode struct {
	UserID              string
	ClientID            string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	ExpiresAt           time.Time
}

type CodeStore struct {
	mu    sync.Mutex
	codes map[string]AuthCode
}

func NewCodeStore() *CodeStore {
	return &CodeStore{codes: make(map[string]AuthCode)}
}

func newRandomToken(byteLen int) (string, error) {
	buffer := make([]byte, byteLen)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// Issue stores a single-use authorization code and returns its value.
func (s *CodeStore) Issue(code AuthCode, now time.Time) (string, error) {
	value, err := newRandomToken(32)
	if err != nil {
		return "", err
	}
	code.ExpiresAt = now.Add(codeTTL)

	s.mu.Lock()
	defer s.mu.Unlock()
	for key, existing := range s.codes {
		if now.After(existing.ExpiresAt) {
			delete(s.codes, key)
		}
	}
	s.codes[value] = code
	return value, nil
}

// Consume returns and deletes the code. ok is false when missing or expired.
func (s *CodeStore) Consume(value string, now time.Time) (AuthCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	code, ok := s.codes[value]
	if !ok {
		return AuthCode{}, false
	}
	delete(s.codes, value)
	if now.After(code.ExpiresAt) {
		return AuthCode{}, false
	}
	return code, true
}

// VerifyPKCE checks an S256 code_verifier against the stored code_challenge.
func VerifyPKCE(method string, challenge string, verifier string) bool {
	if method != "S256" {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// NewClientID mints a public client identifier for dynamic client registration.
func NewClientID() (string, error) {
	value, err := newRandomToken(24)
	if err != nil {
		return "", err
	}
	return "kantor_mcp_" + value, nil
}
