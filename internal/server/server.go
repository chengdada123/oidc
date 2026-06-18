package server

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"oidc-bridge/internal/config"
	"oidc-bridge/internal/db"
	oidcbridge "oidc-bridge/internal/oidc"
	"oidc-bridge/internal/web"
)

type Server struct {
	cfg      *config.Config
	db       *db.Store
	oidc     *oidcbridge.Provider
	renderer *web.Renderer
	sessions sessions.Store
}

func NewRouter(cfg *config.Config, store *db.Store, provider *oidcbridge.Provider, renderer *web.Renderer, sessionStore sessions.Store) http.Handler {
	s := &Server{cfg: cfg, db: store, oidc: provider, renderer: renderer, sessions: sessionStore}
	r := chi.NewRouter()
	r.Get("/", s.handleHome)
	r.Get("/oidc/start", s.handleOIDCStart)
	r.Get("/oidc/callback", s.handleOIDCCallback)
	r.Get("/logout", s.handleLogout)
	r.Get("/.well-known/openid-configuration", s.handleProviderDiscovery)
	r.Get("/oauth/jwks.json", s.handleProviderJWKS)
	r.Get("/oauth/authorize", s.handleProviderAuthorize)
	r.Post("/oauth/authorize/select", s.requireUser(s.handleProviderAuthorizeSelect))
	r.Post("/oauth/token", s.handleProviderToken)
	r.Get("/oauth/userinfo", s.handleProviderUserinfo)
	r.Get("/dashboard", s.requireUser(s.handleDashboard))
	r.Post("/dashboard/emails", s.requireUser(s.handleCreateUserEmail))
	r.Post("/dashboard/emails/{id}/delete", s.requireUser(s.handleDeleteUserEmail))
	r.Get("/auth/start", s.requireUser(s.handleStartTargetLogin))
	r.Get("/admin/login", s.handleAdminLoginPage)
	r.Post("/admin/login", s.handleAdminLogin)
	r.Get("/admin/logout", s.handleAdminLogout)
	r.Get("/admin", s.requireAdmin(s.handleAdminPage))
	r.Post("/admin/settings/email-limit", s.requireAdmin(s.handleUpdateEmailLimit))
	r.Post("/admin/domains", s.requireAdmin(s.handleCreateDomain))
	r.Post("/admin/domains/{id}", s.requireAdmin(s.handleUpdateDomain))
	r.Post("/admin/domains/{id}/delete", s.requireAdmin(s.handleDeleteDomain))
	r.Post("/admin/targets", s.requireAdmin(s.handleCreateTarget))
	r.Post("/admin/targets/{id}", s.requireAdmin(s.handleUpdateTarget))
	r.Post("/admin/targets/{id}/reset-secret", s.requireAdmin(s.handleResetTargetSecret))
	r.Post("/admin/targets/{id}/delete", s.requireAdmin(s.handleDeleteTarget))
	r.Post("/admin/user-emails/{id}/delete", s.requireAdmin(s.handleAdminDeleteUserEmail))
	r.Post("/admin/users/{id}/disable", s.requireAdmin(s.handleAdminDisableUser))
	r.Post("/admin/users/{id}/enable", s.requireAdmin(s.handleAdminEnableUser))
	return r
}

func (s *Server) getSession(r *http.Request) (*sessions.Session, error) {
	return s.sessions.Get(r, "oidc-bridge")
}
func parseInt64(v string) int64 { n, _ := strconv.ParseInt(v, 10, 64); return n }
func (s *Server) currentUser(r *http.Request) (*db.User, error) {
	sess, err := s.getSession(r)
	if err != nil {
		return nil, err
	}
	sub, _ := sess.Values["user_sub"].(string)
	if sub == "" {
		return nil, sql.ErrNoRows
	}
	user, err := s.db.GetUserBySub(sub)
	if err != nil {
		return nil, err
	}
	if user.Disabled {
		return nil, sql.ErrNoRows
	}
	return user, nil
}
func (s *Server) requireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := s.currentUser(r); err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next(w, r)
	}
}
func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.getSession(r)
		if err != nil {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		if ok, _ := sess.Values["admin"].(bool); !ok {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if _, err := s.currentUser(r); err == nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	if s.oidc != nil {
		http.Redirect(w, r, "/oidc/start", http.StatusFound)
		return
	}
	s.renderer.Render(w, "home.html", HomeData{Title: "OIDC Bridge", OIDCReady: false})
}

func (s *Server) handleOIDCStart(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		http.Error(w, "vps8 oidc is not configured yet", http.StatusServiceUnavailable)
		return
	}
	state, err := oidcbridge.NewState()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	sess, _ := s.getSession(r)
	sess.Values["oidc_state"] = state
	_ = sess.Save(r, w)
	http.Redirect(w, r, s.oidc.AuthCodeURL(state), http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if s.oidc == nil {
		http.Error(w, "vps8 oidc is not configured yet", http.StatusServiceUnavailable)
		return
	}
	sess, _ := s.getSession(r)
	state, _ := sess.Values["oidc_state"].(string)
	if r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != state {
		http.Error(w, "invalid state", 400)
		return
	}
	token, err := s.oidc.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	claims, err := s.oidc.VerifyAndFetchUser(r.Context(), token)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	user, err := s.db.UpsertUser(claims.Sub, claims.Email, claims.Name)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	sess.Values["user_sub"] = user.Sub
	delete(sess.Values, "oidc_state")
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.getSession(r)
	delete(sess.Values, "user_sub")
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	domains, _ := s.db.ListDomains()
	emails, _ := s.db.ListUserEmails(user.ID)
	targets, _ := s.db.ListTargets()
	limit, _ := s.db.GetEmailLimitPerUser()
	targetNames := map[int64]string{}
	for _, t := range targets {
		targetNames[t.ID] = t.Name
	}
	var enabledDomains []db.Domain
	for _, d := range domains {
		if d.Enabled && d.TargetID > 0 {
			enabledDomains = append(enabledDomains, d)
		}
	}
	var rows []DashboardEmailRow
	for _, e := range emails {
		d, err := s.db.GetDomain(e.DomainID)
		if err != nil {
			continue
		}
		rows = append(rows, DashboardEmailRow{ID: e.ID, Email: e.Email, Note: e.Note, Enabled: e.Enabled, Domain: d.Domain, TargetName: targetNames[d.TargetID]})
	}
	msg := strings.TrimSpace(r.URL.Query().Get("msg"))
	s.renderer.Render(w, "dashboard.html", DashboardData{Title: "Dashboard", User: user, Domains: enabledDomains, Emails: rows, EmailLimit: limit, Error: msg})
}

var localPartRe = regexp.MustCompile(`^[a-zA-Z0-9._%+-]{1,64}$`)

func (s *Server) handleCreateUserEmail(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	_ = r.ParseForm()
	limit, _ := s.db.GetEmailLimitPerUser()
	count, _ := s.db.CountUserEmails(user.ID)
	if count >= limit {
		http.Redirect(w, r, "/dashboard?msg=已达到邮箱数量上限", http.StatusFound)
		return
	}
	domainID := parseInt64(r.FormValue("domain_id"))
	local := strings.ToLower(strings.TrimSpace(r.FormValue("local_part")))
	note := strings.TrimSpace(r.FormValue("note"))
	if !localPartRe.MatchString(local) {
		http.Redirect(w, r, "/dashboard?msg=邮箱前缀格式不正确", http.StatusFound)
		return
	}
	domain, err := s.db.GetDomain(domainID)
	if err != nil || !domain.Enabled || domain.TargetID <= 0 {
		http.Redirect(w, r, "/dashboard?msg=域名未正确绑定应用", http.StatusFound)
		return
	}
	email := local + "@" + domain.Domain
	if err := s.db.CreateUserEmail(user.ID, domain.ID, local, email, note, true); err != nil {
		http.Redirect(w, r, "/dashboard?msg=创建失败，邮箱可能已被占用", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/dashboard?msg=邮箱创建成功", http.StatusFound)
}

func (s *Server) handleDeleteUserEmail(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	_ = s.db.DeleteUserEmail(parseInt64(chi.URLParam(r, "id")), user.ID)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (s *Server) handleStartTargetLogin(w http.ResponseWriter, r *http.Request) {
	user, _ := s.currentUser(r)
	emailID := parseInt64(r.URL.Query().Get("email_id"))
	email, err := s.db.GetUserEmail(emailID, user.ID)
	if err != nil || !email.Enabled {
		http.Redirect(w, r, "/dashboard?msg=邮箱不存在或不属于当前用户", http.StatusFound)
		return
	}
	domain, err := s.db.GetDomain(email.DomainID)
	if err != nil || domain.TargetID <= 0 {
		http.Redirect(w, r, "/dashboard?msg=该邮箱所属域名未绑定应用", http.StatusFound)
		return
	}
	target, err := s.db.GetTarget(domain.TargetID)
	if err != nil || !target.Enabled {
		http.Redirect(w, r, "/dashboard?msg=对应应用不存在或已禁用", http.StatusFound)
		return
	}
	if target.ClientID == "" || target.RedirectURL == "" {
		http.Redirect(w, r, "/dashboard?msg=对应应用OIDC配置不完整", http.StatusFound)
		return
	}
	if strings.TrimSpace(target.LoginURL) == "" {
		target.LoginURL = target.RedirectURL
		if strings.HasSuffix(target.LoginURL, "/callback") {
			target.LoginURL = strings.TrimSuffix(target.LoginURL, "/callback")
		}
		_ = s.db.UpdateTarget(*target)
	}
	if target.LoginURL == "" {
		http.Redirect(w, r, "/dashboard?msg=对应应用OIDC登录入口为空", http.StatusFound)
		return
	}
	sess, _ := s.getSession(r)
	sess.Values["preferred_email:"+target.ClientID] = email.ID
	_ = sess.Save(r, w)
	http.Redirect(w, r, target.LoginURL, http.StatusFound)
}

func urlQueryEscape(v string) string {
	r := strings.NewReplacer("%", "%25", " ", "%20", "@", "%40", "+", "%2B", "/", "%2F", "?", "%3F", "&", "%26", "=", "%3D")
	return r.Replace(v)
}

func (s *Server) handleAdminLoginPage(w http.ResponseWriter, r *http.Request) {
	s.renderer.Render(w, "admin_login.html", AdminLoginData{Title: "Admin Login"})
}
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.FormValue("password") != s.cfg.AdminPassword {
		w.WriteHeader(http.StatusUnauthorized)
		s.renderer.Render(w, "admin_login.html", AdminLoginData{Title: "Admin Login", Error: "密码不正确"})
		return
	}
	sess, _ := s.getSession(r)
	sess.Values["admin"] = true
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/admin", http.StatusFound)
}
func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	sess, _ := s.getSession(r)
	delete(sess.Values, "admin")
	_ = sess.Save(r, w)
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}
