package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"go-cli-auth/internal/auth"
	"go-cli-auth/internal/config"
	"go-cli-auth/internal/db"

	"github.com/pquerna/otp/totp"
)

func main() {
	_ = os.Remove("app.db")

	cfg := config.LoadConfig()
	database, err := db.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("DB Init failed: %v", err)
	}
	defer database.Close()

	fmt.Println("--- Testing Phase 3: TOTP 2FA ---")

	// 1. Register User
	user, err := auth.RegisterUser(database, "charlie", "password123")
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}
	fmt.Printf("[✓] User '%s' registered\n", user.Username)

	// 2. Generate 2FA Secret Key
	setup, err := auth.GenerateTOTPKey(user.Username)
	if err != nil {
		log.Fatalf("Failed to generate TOTP key: %v", err)
	}
	fmt.Printf("[✓] Generated Secret Key: %s\n", setup.Secret)
	fmt.Printf("[✓] OTP URI (for Authenticator App): %s\n", setup.URL)

	// 3. Enable 2FA in Database
	err = auth.Enable2FA(database, user.ID, setup.Secret)
	if err != nil {
		log.Fatalf("Failed to enable 2FA: %v", err)
	}
	fmt.Println("[✓] 2FA enabled in database for user")

	// 4. Simulate Authenticator App generating a valid 6-digit passcode
	validCode, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}
	fmt.Printf("[*] Current 30-second TOTP code from app: %s\n", validCode)

	// 5. Validate valid code
	if auth.ValidateTOTPCode(validCode, setup.Secret) {
		fmt.Println("[✓] Passcode correctly validated!")
	} else {
		log.Fatal("Validation failed on valid code")
	}

	// 6. Validate invalid code
	if !auth.ValidateTOTPCode("000000", setup.Secret) {
		fmt.Println("[✓] Invalid passcode '000000' correctly rejected!")
	}

	// 7. Disable 2FA
	err = auth.Disable2FA(database, user.ID)
	if err != nil {
		log.Fatalf("Failed to disable 2FA: %v", err)
	}
	fmt.Println("[✓] 2FA successfully disabled")
}