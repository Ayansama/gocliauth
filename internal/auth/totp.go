package auth

import (
	"database/sql"
	"errors"

	"github.com/pquerna/otp/totp"
)

var ErrInvalidTOTP = errors.New("invalid or expired 2FA passcode")

// TOTPSetup holds the secret key and the otpauth URI for app pairing.
type TOTPSetup struct {
	Secret string
	URL    string
}

// GenerateTOTPKey creates a new secret key for a user.
func GenerateTOTPKey(username string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GoCLIAuthSystem",
		AccountName: username,
	})
	if err != nil {
		return nil, err
	}

	return &TOTPSetup{
		Secret: key.Secret(),
		URL:    key.URL(),
	}, nil
}

// ValidateTOTPCode checks if the 6-digit code matches the secret key for the current time window.
func ValidateTOTPCode(code, secret string) bool {
	return totp.Validate(code, secret)
}

// Enable2FA saves the TOTP secret to the user's account and sets totp_enabled = 1.
func Enable2FA(db *sql.DB, userID int64, secret string) error {
	_, err := db.Exec(
		"UPDATE users SET totp_secret = ?, totp_enabled = 1 WHERE id = ?",
		secret, userID,
	)
	return err
}

// Disable2FA clears the TOTP secret and sets totp_enabled = 0.
func Disable2FA(db *sql.DB, userID int64) error {
	_, err := db.Exec(
		"UPDATE users SET totp_secret = '', totp_enabled = 0 WHERE id = ?",
		userID,
	)
	return err
}