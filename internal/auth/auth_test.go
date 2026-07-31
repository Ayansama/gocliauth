package auth

import (
	"os"
	"testing"
	"time"

	"go-cli-auth/internal/config"
	"go-cli-auth/internal/db"

	"github.com/pquerna/otp/totp"
)

func TestRegisterAndAuthenticate(t *testing.T) {
	dbFile := "test_auth.db"
	os.Remove(dbFile)
	defer os.Remove(dbFile)

	cfg := config.LoadConfig()
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	_, err = RegisterUser(database, "testuser", "password123")
	if err != nil {
		t.Fatalf("Expected registration success, got: %v", err)
	}

	_, err = RegisterUser(database, "testuser", "password123")
	if err != ErrUserExists {
		t.Fatalf("Expected ErrUserExists, got: %v", err)
	}

	authUser, err := AuthenticateUser(database, cfg, "testuser", "password123")
	if err != nil || authUser.Username != "testuser" {
		t.Fatalf("Expected auth success, got err: %v", err)
	}

	_, err = AuthenticateUser(database, cfg, "testuser", "wrongpassword")
	if err != ErrInvalidCreds {
		t.Fatalf("Expected ErrInvalidCreds, got: %v", err)
	}
}

func TestAccountLockout(t *testing.T) {
	dbFile := "test_lockout.db"
	os.Remove(dbFile)
	defer os.Remove(dbFile)

	cfg := config.LoadConfig()
	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	_, err = RegisterUser(database, "lockuser", "password123")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	for i := 0; i < cfg.LockoutThreshold-1; i++ {
		_, err := AuthenticateUser(database, cfg, "lockuser", "wrongpass")
		if err != ErrInvalidCreds {
			t.Fatalf("Expected ErrInvalidCreds on attempt %d, got: %v", i+1, err)
		}
	}

	_, err = AuthenticateUser(database, cfg, "lockuser", "wrongpass")
	if err != ErrAccountLocked {
		t.Fatalf("Expected ErrAccountLocked, got: %v", err)
	}

	_, err = AuthenticateUser(database, cfg, "lockuser", "password123")
	if err != ErrAccountLocked {
		t.Fatalf("Expected account to remain locked even with correct password, got: %v", err)
	}
}

func TestTOTPFlow(t *testing.T) {
	dbFile := "test_totp.db"
	os.Remove(dbFile)
	defer os.Remove(dbFile)

	database, err := db.InitDB(dbFile)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	defer database.Close()

	user, err := RegisterUser(database, "totpuser", "password123")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	setup, err := GenerateTOTPKey(user.Username)
	if err != nil {
		t.Fatalf("Failed to generate TOTP key: %v", err)
	}

	err = Enable2FA(database, user.ID, setup.Secret)
	if err != nil {
		t.Fatalf("Failed to enable 2FA: %v", err)
	}

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("Failed to generate TOTP code: %v", err)
	}

	if !ValidateTOTPCode(code, setup.Secret) {
		t.Fatalf("Expected TOTP code validation to succeed")
	}

	if ValidateTOTPCode("000000", setup.Secret) {
		t.Fatalf("Expected invalid TOTP code to fail validation")
	}

	err = Disable2FA(database, user.ID)
	if err != nil {
		t.Fatalf("Failed to disable 2FA: %v", err)
	}
}