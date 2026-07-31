package cli

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"go-cli-auth/internal/auth"
	"go-cli-auth/internal/config"
	"go-cli-auth/internal/session"

	"github.com/chzyer/readline"
)

type AppCLI struct {
	db          *sql.DB
	cfg         *config.Config
	currentUser *auth.User
	session     *session.Session
}

func NewAppCLI(db *sql.DB, cfg *config.Config) *AppCLI {
	return &AppCLI{
		db:  db,
		cfg: cfg,
	}
}

var loggedOutCompleter = readline.NewPrefixCompleter(
	readline.PcItem("register"),
	readline.PcItem("login"),
	readline.PcItem("help"),
	readline.PcItem("exit"),
)

var loggedInCompleter = readline.NewPrefixCompleter(
	readline.PcItem("whoami"),
	readline.PcItem("enable-2fa"),
	readline.PcItem("disable-2fa"),
	readline.PcItem("logout"),
	readline.PcItem("help"),
	readline.PcItem("exit"),
)

func (cli *AppCLI) Start() {
	fmt.Println("========================================")
	fmt.Println("  Welcome to CLI Auth System!!!")
	fmt.Println("  Type 'help' for available commands.")
	fmt.Println("========================================")

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[32m[guest]\033[0m> ",
		HistoryFile:     ".cli_history",
		AutoComplete:    loggedOutCompleter,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("CLI Initiation Error: %v\n", err)
		return
	}
	defer rl.Close()

	for {
		if cli.currentUser == nil {
			rl.SetPrompt("\033[32m[guest]\033[0m> ")
			rl.Config.AutoComplete = loggedOutCompleter
		} else {
			rl.SetPrompt(fmt.Sprintf("\033[36m[%s]\033[0m> ", cli.currentUser.Username))
			rl.Config.AutoComplete = loggedInCompleter
		}

		line, err := rl.Readline()
		if err != nil {
			fmt.Println("\nGoodbye!")
			return
		}

		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		cli.handleCommand(cmd)
	}
}

func (cli *AppCLI) handleCommand(cmd string) {
	if cli.currentUser == nil {
		switch cmd {
		case "register":
			cli.handleRegister()
		case "login":
			cli.handleLogin()
		case "help":
			cli.printHelp()
		case "exit":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Printf("Unknown command: '%s'. Type 'help' for available commands.\n", cmd)
		}
	} else {
		if _, err := session.ValidateSession(cli.db, cli.session.ID); err != nil {
			fmt.Println("\n[!] Your session has expired. Please log in again.")
			cli.currentUser = nil
			cli.session = nil
			return
		}

		switch cmd {
		case "whoami":
			cli.displayUserDetails()
		case "enable-2fa":
			cli.handleEnable2FA()
		case "disable-2fa":
			cli.handleDisable2FA()
		case "logout":
			cli.handleLogout()
		case "help":
			cli.printHelp()
		case "exit":
			fmt.Println("Goodbye!")
			os.Exit(0)
		default:
			fmt.Printf("Unknown command: '%s'. Type 'help' for available commands.\n", cmd)
		}
	}
}

func (cli *AppCLI) printHelp() {
	fmt.Println("\nAvailable Commands:")
	if cli.currentUser == nil {
		fmt.Println("  register   - Create a new account")
		fmt.Println("  login      - Log in to your account")
		fmt.Println("  help       - Show available commands")
		fmt.Println("  exit       - Quit the application")
	} else {
		fmt.Println("  whoami     - View account and session details")
		fmt.Println("  enable-2fa - Enable TOTP Multi-Factor Authentication")
		fmt.Println("  disable-2fa- Disable TOTP Multi-Factor Authentication")
		fmt.Println("  logout     - Log out of current session")
		fmt.Println("  help       - Show available commands")
		fmt.Println("  exit       - Quit the application")
	}
	fmt.Println()
}

func (cli *AppCLI) handleRegister() {
	rl, _ := readline.NewEx(&readline.Config{Prompt: "Enter Username: "})
	defer rl.Close()
	username, _ := rl.Readline()
	username = strings.TrimSpace(username)

	if username == "" {
		fmt.Println("Username cannot be empty.")
		return
	}

	pwRl, _ := readline.NewEx(&readline.Config{Prompt: "Enter Password: "})
	defer pwRl.Close()
	password, err := pwRl.ReadPassword("Enter Password: ")
	if err != nil || len(password) == 0 {
		fmt.Println("Password cannot be empty.")
		return
	}

	_, err = auth.RegisterUser(cli.db, username, string(password))
	if err != nil {
		fmt.Printf("Registration Error: %v\n", err)
		return
	}

	fmt.Printf("Account '%s' registered successfully! You can now log in.\n", username)
}

func (cli *AppCLI) handleLogin() {
	rl, _ := readline.NewEx(&readline.Config{Prompt: "Enter Username: "})
	defer rl.Close()
	username, _ := rl.Readline()
	username = strings.TrimSpace(username)

	pwRl, _ := readline.NewEx(&readline.Config{Prompt: "Enter Password: "})
	defer pwRl.Close()
	passwordBytes, err := pwRl.ReadPassword("Enter Password: ")
	if err != nil {
		fmt.Println("Error reading password.")
		return
	}

	user, err := auth.AuthenticateUser(cli.db, cli.cfg, username, string(passwordBytes))
	if err != nil {
		fmt.Printf("Login Failed: %v\n", err)
		return
	}

	if user.TOTPEnabled {
		totpRl, _ := readline.NewEx(&readline.Config{Prompt: "Enter 6-digit 2FA Passcode: "})
		defer totpRl.Close()
		totpCode, _ := totpRl.Readline()
		totpCode = strings.TrimSpace(totpCode)

		if !auth.ValidateTOTPCode(totpCode, user.TOTPSecret) {
			fmt.Println("Login Failed: Invalid 2FA passcode.")
			return
		}
	}

	sess, err := session.CreateSession(cli.db, user.ID, cli.cfg.SessionTimeout)
	if err != nil {
		fmt.Printf("Session creation error: %v\n", err)
		return
	}

	cli.currentUser = user
	cli.session = sess

	fmt.Println("\nLogin successful!")
	cli.displayUserDetails()
}

func (cli *AppCLI) handleEnable2FA() {
	if cli.currentUser.TOTPEnabled {
		fmt.Println("2FA is already enabled on your account.")
		return
	}

	setup, err := auth.GenerateTOTPKey(cli.currentUser.Username)
	if err != nil {
		fmt.Printf("Error generating 2FA key: %v\n", err)
		return
	}

	fmt.Println("\n--- Enable 2FA ---")
	fmt.Printf("Secret Key: %s\n", setup.Secret)
	fmt.Printf("OTP URI:    %s\n", setup.URL)
	fmt.Println("Add this Secret Key to Google Authenticator / Authy / 1Password.")

	totpRl, _ := readline.NewEx(&readline.Config{Prompt: "Confirm 6-digit code from app: "})
	defer totpRl.Close()
	code, _ := totpRl.Readline()
	code = strings.TrimSpace(code)

	if !auth.ValidateTOTPCode(code, setup.Secret) {
		fmt.Println("Invalid code! 2FA activation cancelled.")
		return
	}

	err = auth.Enable2FA(cli.db, cli.currentUser.ID, setup.Secret)
	if err != nil {
		fmt.Printf("Failed to activate 2FA: %v\n", err)
		return
	}

	cli.currentUser.TOTPEnabled = true
	cli.currentUser.TOTPSecret = setup.Secret
	fmt.Println("2FA has been successfully enabled!")
}

func (cli *AppCLI) handleDisable2FA() {
	if !cli.currentUser.TOTPEnabled {
		fmt.Println("2FA is not enabled on your account.")
		return
	}

	totpRl, _ := readline.NewEx(&readline.Config{Prompt: "Enter 6-digit code to confirm disable: "})
	defer totpRl.Close()
	code, _ := totpRl.Readline()
	code = strings.TrimSpace(code)

	if !auth.ValidateTOTPCode(code, cli.currentUser.TOTPSecret) {
		fmt.Println("Invalid passcode! Operation cancelled.")
		return
	}

	err := auth.Disable2FA(cli.db, cli.currentUser.ID)
	if err != nil {
		fmt.Printf("Failed to disable 2FA: %v\n", err)
		return
	}

	cli.currentUser.TOTPEnabled = false
	cli.currentUser.TOTPSecret = ""
	fmt.Println("2FA has been successfully disabled.")
}

func (cli *AppCLI) handleLogout() {
	if cli.session != nil {
		_ = session.DeleteSession(cli.db, cli.session.ID)
	}
	fmt.Println("Logged out successfully.")
	cli.currentUser = nil
	cli.session = nil
}

func (cli *AppCLI) displayUserDetails() {
	fmt.Println("----------------------------------------")
	fmt.Println("           CURRENT USER DETAILS          ")
	fmt.Println("----------------------------------------")
	fmt.Printf(" Username:          %s\n", cli.currentUser.Username)
	fmt.Printf(" Registration Date: %s\n", cli.currentUser.CreatedAt.Format(time.RFC1123))

	mfaStatus := "Disabled"
	if cli.currentUser.TOTPEnabled {
		mfaStatus = "Enabled"
	}
	fmt.Printf(" MFA Status:        %s\n", mfaStatus)

	if cli.currentUser.LastLogin != nil {
		fmt.Printf(" Last Login:        %s\n", cli.currentUser.LastLogin.Format(time.RFC1123))
	} else {
		fmt.Println(" Last Login:        First time logging in")
	}

	if cli.session != nil {
		fmt.Printf(" Session Expires:   %s\n", cli.session.ExpiresAt.Format(time.RFC1123))
	}
	fmt.Println("----------------------------------------")
}