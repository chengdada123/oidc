package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

type Domain struct {
	ID          int64
	TargetID    int64
	Domain      string
	Description string
	Enabled     bool
	CreatedAt   string
	UpdatedAt   string
}

type Target struct {
	ID            int64
	Name          string
	ClientID      string
	ClientSecret  string
	LoginURL      string
	RedirectURL   string
	ExtraParams   string
	HandoffSecret string
	Enabled       bool
	CreatedAt     string
	UpdatedAt     string
}

type User struct {
	ID        int64
	Sub       string
	Email     string
	Name      string
	Disabled  bool
	CreatedAt string
	UpdatedAt string
}

type UserEmail struct {
	ID        int64
	UserID    int64
	DomainID  int64
	LocalPart string
	Email     string
	Note      string
	Enabled   bool
	CreatedAt string
	UpdatedAt string
}

type BrokerCode struct {
	ID          int64
	Code        string
	TargetID    int64
	UserID      int64
	UserEmailID int64
	RedirectURI string
	Scope       string
	Nonce       string
	State       string
	CreatedAt   string
	ExpiresAt   string
	UsedAt      string
}

type BrokerDebug struct {
	TargetID    int64
	UserID      int64
	UserEmailID int64
	Email       string
	IssuedAt    string
}

type BrokerToken struct {
	ID          int64
	TokenHash   string
	TargetID    int64
	UserID      int64
	UserEmailID int64
	Scope       string
	CreatedAt   string
	ExpiresAt   string
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS domains (id INTEGER PRIMARY KEY AUTOINCREMENT, target_id INTEGER NOT NULL DEFAULT 0, domain TEXT NOT NULL UNIQUE, description TEXT NOT NULL DEFAULT '', enabled INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, FOREIGN KEY(target_id) REFERENCES targets(id));`,
		`CREATE TABLE IF NOT EXISTS targets (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, client_id TEXT NOT NULL DEFAULT '', client_secret TEXT NOT NULL DEFAULT '', login_url TEXT NOT NULL DEFAULT '', redirect_url TEXT NOT NULL, extra_params TEXT NOT NULL DEFAULT '', handoff_secret TEXT NOT NULL DEFAULT '', enabled INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, sub TEXT NOT NULL UNIQUE, email TEXT NOT NULL, name TEXT NOT NULL DEFAULT '', disabled INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);`,
		`CREATE TABLE IF NOT EXISTS user_emails (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, domain_id INTEGER NOT NULL, local_part TEXT NOT NULL, email TEXT NOT NULL, note TEXT NOT NULL DEFAULT '', enabled INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(email), FOREIGN KEY(user_id) REFERENCES users(id), FOREIGN KEY(domain_id) REFERENCES domains(id));`,
		`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TEXT NOT NULL);`,
		`INSERT OR IGNORE INTO settings (key, value, updated_at) VALUES ('email_limit_per_user', '10', datetime('now'));`,
		`CREATE TABLE IF NOT EXISTS broker_codes (id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT NOT NULL UNIQUE, target_id INTEGER NOT NULL, user_id INTEGER NOT NULL, user_email_id INTEGER NOT NULL, redirect_uri TEXT NOT NULL, scope TEXT NOT NULL, nonce TEXT NOT NULL DEFAULT '', state TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, expires_at TEXT NOT NULL, used_at TEXT NOT NULL DEFAULT '', FOREIGN KEY(target_id) REFERENCES targets(id), FOREIGN KEY(user_id) REFERENCES users(id), FOREIGN KEY(user_email_id) REFERENCES user_emails(id));`,
		`CREATE TABLE IF NOT EXISTS broker_tokens (id INTEGER PRIMARY KEY AUTOINCREMENT, token_hash TEXT NOT NULL UNIQUE, target_id INTEGER NOT NULL, user_id INTEGER NOT NULL, user_email_id INTEGER NOT NULL, scope TEXT NOT NULL, created_at TEXT NOT NULL, expires_at TEXT NOT NULL, FOREIGN KEY(target_id) REFERENCES targets(id), FOREIGN KEY(user_id) REFERENCES users(id), FOREIGN KEY(user_email_id) REFERENCES user_emails(id));`,
	}
	compat := []string{
		`ALTER TABLE targets ADD COLUMN extra_params TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE targets ADD COLUMN handoff_secret TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE targets ADD COLUMN client_id TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE targets ADD COLUMN client_secret TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE targets ADD COLUMN login_url TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE user_emails ADD COLUMN note TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE domains ADD COLUMN target_id INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE users ADD COLUMN disabled INTEGER NOT NULL DEFAULT 0;`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate sqlite: %w", err)
		}
	}
	for _, stmt := range compat {
		_, _ = db.Exec(stmt)
	}
	return nil
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
