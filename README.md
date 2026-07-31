# go-cli-auth

A terminal-based authentication system written in Go. Supports user registration, bcrypt password hashing, TOTP-based two-factor authentication, session management, and account lockout — all backed by SQLite and runnable locally or via Docker.

---

## Features

- **Account registration** with bcrypt-hashed passwords
- **Login with MFA** — optional TOTP second factor (Google Authenticator, Authy, 1Password)
- **Session tokens** — 32-char hex tokens stored in SQLite, validated on every command
- **Account lockout** — configurable failed-attempt threshold and lockout window
- **Interactive readline shell** — tab completion, command history (`.cli_history`)
- **Zero-dependency database** — pure-Go SQLite via `modernc.org/sqlite` (no CGO)
- **Docker-ready** — multi-stage Dockerfile, persistent volume via Docker Compose

---

## Architecture

```
┌──────────────────────────────────────────────────┐
│                    cmd/app/main.go                │
│  LoadConfig → InitDB → NewAppCLI → Start()       │
└───────────────────────┬──────────────────────────┘
                        │
         ┌──────────────▼──────────────┐
         │       internal/cli          │
         │   readline REPL loop        │
         │   state: currentUser/session│
         └──┬──────────┬──────────┬───┘
            │          │          │
    ┌───────▼───┐ ┌────▼────┐ ┌──▼──────────┐
    │ auth pkg  │ │  totp   │ │  session pkg │
    │ Register  │ │Generate │ │ Create       │
    │ Authentic.│ │Validate │ │ Validate     │
    │ Lockout   │ │Enable   │ │ Delete       │
    └───────┬───┘ │Disable  │ └──────┬───────┘
            │     └────┬────┘        │
            └──────────┴─────────────┘
                        │
              ┌─────────▼─────────┐
              │  internal/db      │
              │  InitDB()         │
              │  schema.sql embed │
              └─────────┬─────────┘
                        │
              ┌─────────▼─────────┐
              │    app.db         │
              │  (SQLite)         │
              │  users table      │
              │  sessions table   │
              └───────────────────┘
```

---

## Project Structure

```
go-cli-auth/
├── cmd/
│   └── app/
│       └── main.go          # Entrypoint — wires config, db, and CLI
├── internal/
│   ├── auth/
│   │   ├── auth.go          # RegisterUser, AuthenticateUser, lockout logic
│   │   ├── auth_test.go
│   │   ├── totp.go          # GenerateTOTPKey, ValidateTOTPCode, Enable/Disable2FA
│   ├── cli/
│   │   └── cli.go           # Interactive readline REPL, all command handlers
│   ├── config/
│   │   └── config.go        # Env-var config with defaults
│   ├── db/
│   │   ├── db.go            # InitDB — opens SQLite and applies schema
│   │   └── schema.sql       # Embedded DDL (users + sessions tables)
│   └── session/
│       ├── session.go       # CreateSession, ValidateSession, DeleteSession
│       └── session_test.go
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── go.sum
```

---

## Prerequisites

**Local run:**

- Go 1.26 or later (`go version`),

**Docker run:**

- Docker + Docker Compose

---

## Setup

### Option A — Run locally

```bash
# 1. Clone
git clone https://github.com/your-username/go-cli-auth.git
cd go-cli-auth

# 2. Download dependencies
go mod download

# 3. Run (creates app.db in the working directory)
go run ./cmd/app/main.go
```

### Option B — Docker Compose (Recommended)

````bash
# Build and run the interactive CLI container
docker compose run --rm app

The SQLite database is stored in a named Docker volume (`sqlite_data`) and survives container restarts.

### Option C — Build binary manually

```bash
go build -o auth-cli ./cmd/app/main.go
./auth-cli
````

---

## Configuration

All settings are read from environment variables. If a variable is absent or empty, the default applies.

| Variable                | Default  | Description                                      |
| ----------------------- | -------- | ------------------------------------------------ |
| `DB_PATH`               | `app.db` | Path to the SQLite database file                 |
| `LOCKOUT_THRESHOLD`     | `5`      | Failed login attempts before lockout             |
| `LOCKOUT_DURATION_MINS` | `15`     | How long (minutes) a locked account stays locked |
| `SESSION_TIMEOUT_MINS`  | `15`     | How long (minutes) before a session expires      |

**Example — override via shell:**

```bash
DB_PATH=/tmp/myauth.db LOCKOUT_THRESHOLD=3 go run ./cmd/app/main.go
```

**Example — override in docker-compose.yml:**

```yaml
environment:
  - LOCKOUT_THRESHOLD=3
  - SESSION_TIMEOUT_MINS=30
```

---

## Usage Guide

Once started, you'll see the interactive prompt. The shell provides tab-completion and persists command history to `.cli_history`.

### Guest commands (not logged in)

```
[guest]> register
[guest]> login
[guest]> help
[guest]> exit
```

### Authenticated commands (logged in)

```
[username]> whoami
[username]> enable-2fa
[username]> disable-2fa
[username]> logout
[username]> help
[username]> exit
```

---

### `register` — Create an account

```
[guest]> register
Enter Username: ayan
Enter Password: ••••••••
Account 'ayan' registered successfully! You can now log in.
```

Passwords are hashed with bcrypt (`DefaultCost = 10`) before storage. The plaintext is never persisted.

---

### `login` — Authenticate

```
[guest]> login
Enter Username: ayan
Enter Password: ••••••••
Enter 6-digit 2FA Passcode: 482910   ← only shown if 2FA is enabled

Login successful!
----------------------------------------
           CURRENT USER DETAILS
----------------------------------------
 Username:          ayan
 Registration Date: Thu, 31 Jul 2026 14:00:00 IST
 MFA Status:        Enabled
 Last Login:        First time logging in
 Session Expires:   Thu, 31 Jul 2026 14:15:00 IST
----------------------------------------
```

After 5 failed attempts (configurable), the account is locked for 15 minutes (configurable). A successful login resets the failed-attempt counter.

---

### `whoami` — View account details

```
[ayan]> whoami
```

Displays username, registration date, MFA status, last login timestamp, and session expiry.

---

### `enable-2fa` — Set up TOTP two-factor authentication

```
[ayan]> enable-2fa

--- Enable 2FA ---
Secret Key: JBSWY3DPEHPK3PXP
OTP URI:    otpauth://totp/GoCLIAuthSystem:ayan?secret=JBSWY3DPEHPK3PXP&issuer=GoCLIAuthSystem
Add this Secret Key to Google Authenticator / Authy / 1Password.
Confirm 6-digit code from app: 123456
2FA has been successfully enabled!
```

Steps:

1. Copy the **Secret Key** (or scan-import the OTP URI) into your authenticator app.
2. Enter the 6-digit code the app shows to confirm pairing.
3. From the next login onward, the app code will be required.

---

### `disable-2fa` — Remove TOTP

```
[ayan]> disable-2fa
Enter 6-digit code to confirm disable: 482910
2FA has been successfully disabled.
```

Requires a valid current code to prevent accidental or unauthorized removal.

---

### `logout` — End session

```
[ayan]> logout
Logged out successfully.
```

Deletes the session from the database immediately. The prompt returns to `[guest]`.

---

## Database Schema

The schema is embedded into the binary at compile time via Go's `//go:embed` directive. It is applied automatically on startup with `CREATE TABLE IF NOT EXISTS`, so no manual migration step is needed.

**File:** `internal/db/schema.sql`

```sql
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    username        TEXT     UNIQUE NOT NULL,
    password_hash   TEXT     NOT NULL,
    totp_secret     TEXT     DEFAULT '',
    totp_enabled    BOOLEAN  DEFAULT 0,
    failed_attempts INTEGER  DEFAULT 0,
    lockout_until   DATETIME DEFAULT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login      DATETIME DEFAULT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT     PRIMARY KEY,
    user_id    INTEGER  NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### Schema notes

- `password_hash` — bcrypt digest; plaintext never stored
- `totp_secret` — base32-encoded TOTP key; stored only after successful `enable-2fa` confirmation
- `failed_attempts` / `lockout_until` — reset to `0` / `NULL` on every successful login
- Sessions use a 32-character random hex token as the primary key; `ON DELETE CASCADE` ensures sessions are cleaned up when a user is deleted
- Foreign keys are enabled at the connection level via `?_pragma=foreign_keys(1)`

---

## Running Tests

```bash
go test -v ./...

Tests live in internal/auth/auth_test.go and internal/session/session_test.go. They create temporary isolated SQLite database files during test execution and automatically clean them up upon completion.

---

## Key Dependencies

| Package                      | Purpose                                           |
| ---------------------------- | ------------------------------------------------- |
| `golang.org/x/crypto/bcrypt` | Password hashing                                  |
| `github.com/pquerna/otp`     | TOTP key generation and validation                |
| `github.com/chzyer/readline` | Interactive shell with tab completion and history |
| `modernc.org/sqlite`         | Pure-Go SQLite driver (no CGO required)           |

---

## Security Notes

- Sessions expire after 15 minutes of inactivity (configurable). Expiry is checked on every authenticated command — not just at login.
- Account lockout is time-based: the lockout window resets automatically; no admin action needed.
- TOTP codes are validated with `pquerna/otp`, which enforces the 30-second time window per RFC 6238.
- The `DB_PATH` file should be kept outside web-accessible directories in production.
```
