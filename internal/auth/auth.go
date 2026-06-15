// Package auth provides single-password authentication backed by DB sessions.
package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const cookieName = "vmt_session"

// CookieName is the name of the session cookie.
func CookieName() string { return cookieName }

type Manager struct {
	db  *sql.DB
	ttl time.Duration
}

func New(db *sql.DB, ttl time.Duration) *Manager {
	return &Manager{db: db, ttl: ttl}
}

// IsConfigured reports whether an admin password has been set.
func (m *Manager) IsConfigured() (bool, error) {
	var v string
	err := m.db.QueryRow(`SELECT value FROM settings WHERE key='password_hash'`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v != "", nil
}

// SetPassword stores a bcrypt hash of the given plaintext password.
func (m *Manager) SetPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(
		`INSERT INTO settings(key, value) VALUES('password_hash', ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		string(hash),
	)
	return err
}

// EnsurePassword sets the password from a bootstrap value only if none exists.
// It returns true if it set the password.
func (m *Manager) EnsurePassword(plain string) (bool, error) {
	if plain == "" {
		return false, nil
	}
	configured, err := m.IsConfigured()
	if err != nil || configured {
		return false, err
	}
	return true, m.SetPassword(plain)
}

// Check verifies a plaintext password against the stored hash.
func (m *Manager) Check(plain string) (bool, error) {
	var hash string
	err := m.db.QueryRow(`SELECT value FROM settings WHERE key='password_hash'`).Scan(&hash)
	if err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil, nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession creates a new session and returns its token.
func (m *Manager) CreateSession() (string, time.Time, error) {
	token, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(m.ttl)
	_, err = m.db.Exec(
		`INSERT INTO sessions(token, expires_at) VALUES(?, ?)`,
		token, expires.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// Valid reports whether the token corresponds to a live session.
func (m *Manager) Valid(token string) bool {
	if token == "" {
		return false
	}
	var expires string
	err := m.db.QueryRow(`SELECT expires_at FROM sessions WHERE token=?`, token).Scan(&expires)
	if err != nil {
		return false
	}
	t, err := time.Parse(time.RFC3339, expires)
	if err != nil {
		return false
	}
	if time.Now().After(t) {
		_, _ = m.db.Exec(`DELETE FROM sessions WHERE token=?`, token)
		return false
	}
	return true
}

// Destroy removes a session.
func (m *Manager) Destroy(token string) {
	_, _ = m.db.Exec(`DELETE FROM sessions WHERE token=?`, token)
}

// CleanupExpired removes expired sessions.
func (m *Manager) CleanupExpired() {
	_, _ = m.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
}
