package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"oidc-bridge/internal/db"
)

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
func generateClientID() string     { return "bridge_" + randomHex(12) }
func generateClientSecret() string { return randomHex(32) }
func secretHint(secret string) string {
	if len(secret) <= 8 {
		return secret
	}
	return secret[:8]
}

func (s *Server) handleUpdateEmailLimit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	limit, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("email_limit")))
	if limit > 0 {
		_ = s.db.SetEmailLimitPerUser(limit)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleCreateDomain(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	targetID := parseInt64(r.FormValue("target_id"))
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	description := strings.TrimSpace(r.FormValue("description"))
	enabled := r.FormValue("enabled") == "1"
	if targetID > 0 && domain != "" {
		_ = s.db.CreateDomain(targetID, domain, description, enabled)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleUpdateDomain(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id := parseInt64(chi.URLParam(r, "id"))
	targetID := parseInt64(r.FormValue("target_id"))
	domain := strings.ToLower(strings.TrimSpace(r.FormValue("domain")))
	description := strings.TrimSpace(r.FormValue("description"))
	enabled := r.FormValue("enabled") == "1"
	if id > 0 && targetID > 0 && domain != "" {
		_ = s.db.UpdateDomain(id, targetID, domain, description, enabled)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleDeleteDomain(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(chi.URLParam(r, "id"))
	if id > 0 {
		_ = s.db.DeleteDomain(id)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	secret := generateClientSecret()
	t := db.Target{
		Name:          strings.TrimSpace(r.FormValue("name")),
		ClientID:      generateClientID(),
		ClientSecret:  secret,
		LoginURL:      strings.TrimSpace(r.FormValue("login_url")),
		RedirectURL:   strings.TrimSpace(r.FormValue("redirect_url")),
		ExtraParams:   strings.TrimSpace(r.FormValue("extra_params")),
		HandoffSecret: secretHint(secret),
		Enabled:       r.FormValue("enabled") == "1",
	}
	if t.Name != "" && t.LoginURL != "" && t.RedirectURL != "" {
		_ = s.db.CreateTarget(t)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleUpdateTarget(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	current, _ := s.db.GetTarget(parseInt64(chi.URLParam(r, "id")))
	clientID := strings.TrimSpace(r.FormValue("client_id"))
	if clientID == "" && current != nil {
		clientID = current.ClientID
	}
	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	hint := ""
	if clientSecret == "" && current != nil {
		clientSecret = current.ClientSecret
		hint = current.HandoffSecret
	} else {
		hint = secretHint(clientSecret)
	}
	t := db.Target{
		ID:            parseInt64(chi.URLParam(r, "id")),
		Name:          strings.TrimSpace(r.FormValue("name")),
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		LoginURL:      strings.TrimSpace(r.FormValue("login_url")),
		RedirectURL:   strings.TrimSpace(r.FormValue("redirect_url")),
		ExtraParams:   strings.TrimSpace(r.FormValue("extra_params")),
		HandoffSecret: hint,
		Enabled:       r.FormValue("enabled") == "1",
	}
	if t.ID > 0 && t.Name != "" && t.LoginURL != "" && t.RedirectURL != "" {
		_ = s.db.UpdateTarget(t)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleResetTargetSecret(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(chi.URLParam(r, "id"))
	if id > 0 {
		current, err := s.db.GetTarget(id)
		if err == nil {
			secret := generateClientSecret()
			current.ClientSecret = secret
			current.HandoffSecret = secretHint(secret)
			_ = s.db.UpdateTarget(*current)
		}
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(chi.URLParam(r, "id"))
	if id > 0 {
		_ = s.db.DeleteTarget(id)
	}
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) handleAdminDeleteUserEmail(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(chi.URLParam(r, "id"))
	if id > 0 {
		_, _ = s.db.DB.Exec(`DELETE FROM user_emails WHERE id = ?`, id)
	}
	redirect := "/admin"
	if qs := r.URL.RawQuery; qs != "" {
		redirect += "?" + qs
	}
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (s *Server) handleAdminPage(w http.ResponseWriter, r *http.Request) {
	domains, _ := s.db.ListDomains()
	targets, _ := s.db.ListTargets()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page")))
	if page < 1 {
		page = 1
	}
	const pageSize = 20
	total, _ := s.db.CountAdminUserEmails(q)
	offset := (page - 1) * pageSize
	if total > 0 && offset >= total {
		page = (total + pageSize - 1) / pageSize
		offset = (page - 1) * pageSize
	}
	allUserEmails, _ := s.db.ListAdminUserEmails(q, pageSize, offset)
	limit, _ := s.db.GetEmailLimitPerUser()
	targetNames := map[int64]string{}
	for _, t := range targets {
		targetNames[t.ID] = t.Name
	}
	domainByID := map[int64]db.Domain{}
	var rows []AdminDomainRow
	for _, d := range domains {
		domainByID[d.ID] = d
		rows = append(rows, AdminDomainRow{ID: d.ID, Domain: d.Domain, Description: d.Description, Enabled: d.Enabled, TargetID: d.TargetID, TargetName: targetNames[d.TargetID]})
	}
	var emailRows []AdminUserEmailRow
	for _, e := range allUserEmails {
		user, err := s.db.GetUserByID(e.UserID)
		if err != nil {
			continue
		}
		domain, ok := domainByID[e.DomainID]
		if !ok {
			continue
		}
		emailRows = append(emailRows, AdminUserEmailRow{ID: e.ID, UserID: user.ID, UserSub: user.Sub, UserEmail: user.Email, UserName: user.Name, Email: e.Email, LocalPart: e.LocalPart, Note: e.Note, Domain: domain.Domain, TargetName: targetNames[domain.TargetID], Enabled: e.Enabled})
	}
	pages := 0
	if total > 0 {
		pages = (total + pageSize - 1) / pageSize
	}
	prevPage := 0
	if page > 1 {
		prevPage = page - 1
	}
	nextPage := 0
	if pages > 0 && page < pages {
		nextPage = page + 1
	}
	s.renderer.Render(w, "admin.html", AdminPageData{Title: "Admin", BaseURL: strings.TrimRight(s.cfg.BaseURL, "/"), Domains: rows, Targets: targets, UserEmails: emailRows, EmailLimit: limit, UserEmailQuery: q, UserEmailPage: page, UserEmailPages: pages, UserEmailTotal: total, UserEmailPrev: prevPage, UserEmailNext: nextPage})
}
