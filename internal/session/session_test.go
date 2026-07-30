package session

import (
	"os"
	"testing"
	"time"

	"go-cli-auth/internal/db"
)

func TestSessionLifecycle(t *testing.T) {
	os.Remove("test_session.db")
	defer os.Remove("test_session.db")

	database, err := db.InitDB("test_session.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	sess, err := CreateSession(database, 1, 10*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	valSess, err := ValidateSession(database, sess.ID)
	if err != nil || valSess.UserID != 1 {
		t.Fatalf("Expected valid session, got err: %v", err)
	}
}