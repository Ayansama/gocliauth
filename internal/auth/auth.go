package auth

import (
	"database/sql"
	"errors"
	"time"

	"go-cli-auth/internal/config"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists    = errors.New("username already exists")
	ErrInvalidCreds  = errors.New("invalid username or password")
	ErrAccountLocked = errors.New("account is locked due to too many failed login attempts")
)

type User struct {
	ID             int64
	Username       string
	PasswordHash   string
	TOTPSecret     string
	TOTPEnabled    bool
	FailedAttempts int
	LockoutUntil   *time.Time
	CreatedAt      time.Time
	LastLogin      *time.Time
}

func RegisterUser(db *sql.DB, username, password string) (*User, error) {
	var exists int
	err := db.QueryRow("SELECT COUNT(1) FROM users WHERE username = ?", username).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, ErrUserExists
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	res, err := db.Exec(
		"INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?)",
		username, string(hashedBytes), now,
	)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: string(hashedBytes),
		CreatedAt:    now,
	}, nil
}

// AuthenticateUser verifies credentials, enforces account lockout thresholds, and updatinng last_login.
func AuthenticateUser(db *sql.DB, cfg *config.Config, username, password string) (*User, error) {
	var user User
	var lockoutUntil sql.NullTime
	var lastLogin sql.NullTime

	query := `SELECT id, username, password_hash, totp_secret, totp_enabled, failed_attempts, lockout_until, created_at, last_login 
	          FROM users WHERE username = ?`
	err := db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.TOTPSecret,
		&user.TOTPEnabled,
		&user.FailedAttempts,
		&lockoutUntil,
		&user.CreatedAt,
		&lastLogin,
	)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidCreds
	} else if err != nil {
		return nil, err
	}

	if lockoutUntil.Valid {
		user.LockoutUntil = &lockoutUntil.Time
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	//Checking if account is currently locked
	if user.LockoutUntil != nil && time.Now().Before(*user.LockoutUntil) {
		return nil, ErrAccountLocked
	}

	//Validating password hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Incorrect password entered: increment failed attempts
		newFailed := user.FailedAttempts + 1
		var newLockout interface{} = nil

		if newFailed >= cfg.LockoutThreshold {
			lockoutTime := time.Now().Add(cfg.LockoutDuration)
			newLockout = lockoutTime
		}

		_, _ = db.Exec(
			"UPDATE users SET failed_attempts = ?, lockout_until = ? WHERE id = ?",
			newFailed, newLockout, user.ID,
		)

		if newFailed >= cfg.LockoutThreshold {
			return nil, ErrAccountLocked
		}
		return nil, ErrInvalidCreds
	}

	//Successful login: reset failed attempts, clear lockout, update last_login timestamp
	now := time.Now()
	_, err = db.Exec(
		"UPDATE users SET failed_attempts = 0, lockout_until = NULL, last_login = ? WHERE id = ?",
		now, user.ID,
	)
	if err != nil {
		return nil, err
	}
	user.LastLogin = &now
	user.FailedAttempts = 0
	user.LockoutUntil = nil

	return &user, nil
}