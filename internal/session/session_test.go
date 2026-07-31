package session

import (
	"os"
	"testing"
	"time"

	"go-cli-auth/internal/db"
)

func TestSessionLifecycle(t *testing.T) {
	dbFile := "test_session.db"
	os.Remove(dbFile)
	defer os.Remove(dbFile)

	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	res, err := database.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", "sessuser", "hash")
	if err != nil {
		t.Fatalf("Failed to create user record for foreign key test: %v", err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to fetch user ID: %v", err)
	}

	sess, err := CreateSession(database, userID, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	valSess, err := ValidateSession(database, sess.ID)
	if err != nil || valSess.UserID != userID {
		t.Fatalf("Expected valid session, got err: %v", err)
	}

	err = DeleteSession(database, sess.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	_, err = ValidateSession(database, sess.ID)
	if err != ErrSessionNotFound {
		t.Fatalf("Expected ErrSessionNotFound after deletion, got: %v", err)
	}

	expSess, err := CreateSession(database, userID, -1*time.Second)
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	_, err = ValidateSession(database, expSess.ID)
	if err != ErrSessionExpired {
		t.Fatalf("Expected ErrSessionExpired, got: %v", err)
	}
}