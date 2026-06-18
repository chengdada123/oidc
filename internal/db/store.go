package db

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Store) UpsertUser(sub, email, name string) (*User, error) {
	var existing User
	err := s.DB.QueryRow(`SELECT id, sub, email, name, created_at, updated_at FROM users WHERE sub = ?`, sub).Scan(&existing.ID, &existing.Sub, &existing.Email, &existing.Name, &existing.CreatedAt, &existing.UpdatedAt)
	if err == nil {
		ts := now()
		_, err = s.DB.Exec(`UPDATE users SET email = ?, name = ?, updated_at = ? WHERE id = ?`, email, name, ts, existing.ID)
		if err != nil {
			return nil, err
		}
		existing.Email, existing.Name, existing.UpdatedAt = email, name, ts
		return &existing, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	ts := now()
	res, err := s.DB.Exec(`INSERT INTO users (sub, email, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, sub, email, name, ts, ts)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Sub: sub, Email: email, Name: name, CreatedAt: ts, UpdatedAt: ts}, nil
}
func (s *Store) GetUserBySub(sub string) (*User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, sub, email, name, created_at, updated_at FROM users WHERE sub = ?`, sub).Scan(&u.ID, &u.Sub, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
func (s *Store) GetUserByID(id int64) (*User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, sub, email, name, created_at, updated_at FROM users WHERE id = ?`, id).Scan(&u.ID, &u.Sub, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) ListDomains() ([]Domain, error) {
	rows, err := s.DB.Query(`SELECT id, target_id, domain, description, enabled, created_at, updated_at FROM domains ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Domain
	for rows.Next() {
		var d Domain
		var enabled int
		if err := rows.Scan(&d.ID, &d.TargetID, &d.Domain, &d.Description, &enabled, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.Enabled = enabled == 1
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *Store) GetDomain(id int64) (*Domain, error) {
	var d Domain
	var enabled int
	err := s.DB.QueryRow(`SELECT id, target_id, domain, description, enabled, created_at, updated_at FROM domains WHERE id = ?`, id).Scan(&d.ID, &d.TargetID, &d.Domain, &d.Description, &enabled, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.Enabled = enabled == 1
	return &d, nil
}
func (s *Store) CreateDomain(targetID int64, domain, description string, enabled bool) error {
	_, err := s.DB.Exec(`INSERT INTO domains (target_id, domain, description, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, targetID, domain, description, boolInt(enabled), now(), now())
	return err
}
func (s *Store) UpdateDomain(id, targetID int64, domain, description string, enabled bool) error {
	_, err := s.DB.Exec(`UPDATE domains SET target_id = ?, domain = ?, description = ?, enabled = ?, updated_at = ? WHERE id = ?`, targetID, domain, description, boolInt(enabled), now(), id)
	return err
}
func (s *Store) DeleteDomain(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM domains WHERE id = ?`, id)
	return err
}

func (s *Store) ListTargets() ([]Target, error) {
	rows, err := s.DB.Query(`SELECT id, name, client_id, client_secret, login_url, redirect_url, extra_params, handoff_secret, enabled, created_at, updated_at FROM targets ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Target
	for rows.Next() {
		var t Target
		var enabled int
		if err := rows.Scan(&t.ID, &t.Name, &t.ClientID, &t.ClientSecret, &t.LoginURL, &t.RedirectURL, &t.ExtraParams, &t.HandoffSecret, &enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Enabled = enabled == 1
		out = append(out, t)
	}
	return out, rows.Err()
}
func (s *Store) GetTarget(id int64) (*Target, error) {
	var t Target
	var enabled int
	err := s.DB.QueryRow(`SELECT id, name, client_id, client_secret, login_url, redirect_url, extra_params, handoff_secret, enabled, created_at, updated_at FROM targets WHERE id = ?`, id).Scan(&t.ID, &t.Name, &t.ClientID, &t.ClientSecret, &t.LoginURL, &t.RedirectURL, &t.ExtraParams, &t.HandoffSecret, &enabled, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Enabled = enabled == 1
	return &t, nil
}
func (s *Store) GetTargetByClientID(clientID string) (*Target, error) {
	var t Target
	var enabled int
	err := s.DB.QueryRow(`SELECT id, name, client_id, client_secret, login_url, redirect_url, extra_params, handoff_secret, enabled, created_at, updated_at FROM targets WHERE client_id = ?`, clientID).Scan(&t.ID, &t.Name, &t.ClientID, &t.ClientSecret, &t.LoginURL, &t.RedirectURL, &t.ExtraParams, &t.HandoffSecret, &enabled, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Enabled = enabled == 1
	return &t, nil
}
func (s *Store) CreateTarget(t Target) error {
	_, err := s.DB.Exec(`INSERT INTO targets (name, client_id, client_secret, login_url, redirect_url, extra_params, handoff_secret, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, t.Name, t.ClientID, t.ClientSecret, t.LoginURL, t.RedirectURL, t.ExtraParams, t.HandoffSecret, boolInt(t.Enabled), now(), now())
	return err
}
func (s *Store) UpdateTarget(t Target) error {
	_, err := s.DB.Exec(`UPDATE targets SET name = ?, client_id = ?, client_secret = ?, login_url = ?, redirect_url = ?, extra_params = ?, handoff_secret = ?, enabled = ?, updated_at = ? WHERE id = ?`, t.Name, t.ClientID, t.ClientSecret, t.LoginURL, t.RedirectURL, t.ExtraParams, t.HandoffSecret, boolInt(t.Enabled), now(), t.ID)
	return err
}
func (s *Store) DeleteTarget(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM targets WHERE id = ?`, id)
	return err
}

func (s *Store) ListUserEmails(userID int64) ([]UserEmail, error) {
	rows, err := s.DB.Query(`SELECT id, user_id, domain_id, local_part, email, note, enabled, created_at, updated_at FROM user_emails WHERE user_id = ? ORDER BY id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserEmail
	for rows.Next() {
		var e UserEmail
		var enabled int
		if err := rows.Scan(&e.ID, &e.UserID, &e.DomainID, &e.LocalPart, &e.Email, &e.Note, &enabled, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *Store) ListAllUserEmails() ([]UserEmail, error) {
	rows, err := s.DB.Query(`SELECT id, user_id, domain_id, local_part, email, note, enabled, created_at, updated_at FROM user_emails ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserEmail
	for rows.Next() {
		var e UserEmail
		var enabled int
		if err := rows.Scan(&e.ID, &e.UserID, &e.DomainID, &e.LocalPart, &e.Email, &e.Note, &enabled, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		out = append(out, e)
	}
	return out, rows.Err()
}
func adminUserEmailFilter(query string) (string, []any) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", nil
	}
	like := "%" + query + "%"
	return ` WHERE ue.email LIKE ? OR ue.local_part LIKE ? OR ue.note LIKE ? OR u.email LIKE ? OR u.name LIKE ? OR d.domain LIKE ? OR t.name LIKE ?`, []any{like, like, like, like, like, like, like}
}
func (s *Store) CountAdminUserEmails(query string) (int, error) {
	where, args := adminUserEmailFilter(query)
	row := s.DB.QueryRow(`SELECT COUNT(*) FROM user_emails ue JOIN users u ON u.id = ue.user_id JOIN domains d ON d.id = ue.domain_id LEFT JOIN targets t ON t.id = d.target_id`+where, args...)
	var total int
	err := row.Scan(&total)
	return total, err
}
func (s *Store) ListAdminUserEmails(query string, limit, offset int) ([]UserEmail, error) {
	where, args := adminUserEmailFilter(query)
	args = append(args, limit, offset)
	rows, err := s.DB.Query(`SELECT ue.id, ue.user_id, ue.domain_id, ue.local_part, ue.email, ue.note, ue.enabled, ue.created_at, ue.updated_at FROM user_emails ue JOIN users u ON u.id = ue.user_id JOIN domains d ON d.id = ue.domain_id LEFT JOIN targets t ON t.id = d.target_id`+where+` ORDER BY ue.id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserEmail
	for rows.Next() {
		var e UserEmail
		var enabled int
		if err := rows.Scan(&e.ID, &e.UserID, &e.DomainID, &e.LocalPart, &e.Email, &e.Note, &enabled, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Enabled = enabled == 1
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *Store) CountUserEmails(userID int64) (int, error) {
	var n int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM user_emails WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}
func (s *Store) GetUserEmail(id, userID int64) (*UserEmail, error) {
	var e UserEmail
	var enabled int
	err := s.DB.QueryRow(`SELECT id, user_id, domain_id, local_part, email, note, enabled, created_at, updated_at FROM user_emails WHERE id = ? AND user_id = ?`, id, userID).Scan(&e.ID, &e.UserID, &e.DomainID, &e.LocalPart, &e.Email, &e.Note, &enabled, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Enabled = enabled == 1
	return &e, nil
}
func (s *Store) GetUserEmailByID(id int64) (*UserEmail, error) {
	var e UserEmail
	var enabled int
	err := s.DB.QueryRow(`SELECT id, user_id, domain_id, local_part, email, note, enabled, created_at, updated_at FROM user_emails WHERE id = ?`, id).Scan(&e.ID, &e.UserID, &e.DomainID, &e.LocalPart, &e.Email, &e.Note, &enabled, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Enabled = enabled == 1
	return &e, nil
}
func (s *Store) CreateUserEmail(userID, domainID int64, localPart, email, note string, enabled bool) error {
	_, err := s.DB.Exec(`INSERT INTO user_emails (user_id, domain_id, local_part, email, note, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, userID, domainID, localPart, email, note, boolInt(enabled), now(), now())
	return err
}
func (s *Store) DeleteUserEmail(id, userID int64) error {
	_, err := s.DB.Exec(`DELETE FROM user_emails WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) GetEmailLimitPerUser() (int, error) {
	var value string
	err := s.DB.QueryRow(`SELECT value FROM settings WHERE key = 'email_limit_per_user'`).Scan(&value)
	if err != nil {
		return 10, err
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 10, nil
	}
	return n, nil
}
func (s *Store) SetEmailLimitPerUser(limit int) error {
	if limit < 1 {
		return fmt.Errorf("limit must be positive")
	}
	_, err := s.DB.Exec(`INSERT INTO settings (key, value, updated_at) VALUES ('email_limit_per_user', ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`, strconv.Itoa(limit), now())
	return err
}

func expiresAfter(minutes int) string {
	return time.Now().UTC().Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339)
}
func (s *Store) CreateBrokerCode(code string, targetID, userID, userEmailID int64, redirectURI, scope, nonce, state string) error {
	_, err := s.DB.Exec(`INSERT INTO broker_codes (code, target_id, user_id, user_email_id, redirect_uri, scope, nonce, state, created_at, expires_at, used_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '')`, code, targetID, userID, userEmailID, redirectURI, scope, nonce, state, now(), expiresAfter(5))
	return err
}
func (s *Store) GetBrokerCode(code string) (*BrokerCode, error) {
	var b BrokerCode
	err := s.DB.QueryRow(`SELECT id, code, target_id, user_id, user_email_id, redirect_uri, scope, nonce, state, created_at, expires_at, used_at FROM broker_codes WHERE code = ?`, code).Scan(&b.ID, &b.Code, &b.TargetID, &b.UserID, &b.UserEmailID, &b.RedirectURI, &b.Scope, &b.Nonce, &b.State, &b.CreatedAt, &b.ExpiresAt, &b.UsedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
func (s *Store) MarkBrokerCodeUsed(id int64) error {
	_, err := s.DB.Exec(`UPDATE broker_codes SET used_at = ? WHERE id = ?`, now(), id)
	return err
}
func (s *Store) CreateBrokerToken(rawToken string, targetID, userID, userEmailID int64, scope string) error {
	sum := sha256.Sum256([]byte(rawToken))
	_, err := s.DB.Exec(`INSERT INTO broker_tokens (token_hash, target_id, user_id, user_email_id, scope, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, fmt.Sprintf("%x", sum[:]), targetID, userID, userEmailID, scope, now(), expiresAfter(60))
	return err
}
func (s *Store) GetBrokerToken(rawToken string) (*BrokerToken, error) {
	sum := sha256.Sum256([]byte(rawToken))
	var b BrokerToken
	err := s.DB.QueryRow(`SELECT id, token_hash, target_id, user_id, user_email_id, scope, created_at, expires_at FROM broker_tokens WHERE token_hash = ?`, fmt.Sprintf("%x", sum[:])).Scan(&b.ID, &b.TokenHash, &b.TargetID, &b.UserID, &b.UserEmailID, &b.Scope, &b.CreatedAt, &b.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
func (s *Store) GetLatestBrokerDebug(targetID int64) (*BrokerDebug, error) {
	var d BrokerDebug
	err := s.DB.QueryRow(`SELECT bc.target_id, bc.user_id, bc.user_email_id, ue.email, bc.created_at FROM broker_codes bc JOIN user_emails ue ON ue.id = bc.user_email_id WHERE bc.target_id = ? ORDER BY bc.id DESC LIMIT 1`, targetID).Scan(&d.TargetID, &d.UserID, &d.UserEmailID, &d.Email, &d.IssuedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
