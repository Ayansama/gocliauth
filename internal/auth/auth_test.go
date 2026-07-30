package auth

import (
	"os"
	"testing"

	"go-cli-auth/internal/config"
	"go-cli-auth/internal/db"
)

func TestRegisterAndAuthenticate(t *testing.T) {
	os.Remove("test_auth.db")
	defer os.Remove("test_auth.db")

	cfg := config.LoadConfig()
	database, err := db.InitDB("test_auth.db")
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	// Test Register
	user, err := RegisterUser(database, "testuser", "password123")
	if err != nil {
		t.Fatalf("Expected registration success, got: %v", err)
	}

	// Test Duplicate Register
	_, err = RegisterUser(database, "testuser", "password123")
	if err != ErrUserExists {
		t.Fatalf("Expected ErrUserExists, got: %v", err)
	}

	// Test Authenticate Success
	authUser, err := AuthenticateUser(database, cfg, "testuser", "password123")
	if err != nil || authUser.Username != "testuser" {
		t.Fatalf("Expected auth success, got err: %v", err)
	}
}