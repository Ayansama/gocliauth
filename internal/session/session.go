package session

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

var (
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionNotFound = errors.New("session not found")
)

type Session struct {
	ID        string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

//generates a new UUID-style session token and stores it in the database.
func CreateSession(db *sql.DB, userID int64, duration time.Duration) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(duration)

	_, err = db.Exec(
		"INSERT INTO sessions (id, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
		token, userID, now, expiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        token,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}, nil
}

//verifies if a session exists and is not expired.
func ValidateSession(db *sql.DB, sessionID string) (*Session, error) {
	var s Session
	err := db.QueryRow(
		"SELECT id, user_id, created_at, expires_at FROM sessions WHERE id = ?",
		sessionID,
	).Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.ExpiresAt)

	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	} else if err != nil {
		return nil, err
	}

	if time.Now().After(s.ExpiresAt) {
		_ = DeleteSession(db, sessionID)
		return nil, ErrSessionExpired
	}

	return &s, nil
}

//deletes a session from the database.
func DeleteSession(db *sql.DB, sessionID string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}