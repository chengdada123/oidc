package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"oidc-bridge/internal/db"
	oidcbridge "oidc-bridge/internal/oidc"
)

func issuerBase(baseURL string) string { return strings.TrimRight(baseURL, "/") }

func (s *Server) handleProviderDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(oidcbridge.BuildDiscovery(issuerBase(s.cfg.BaseURL)))
}

func (s *Server) handleProviderJWKS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	jwk, err := oidcbridge.BuildJWK(oidcbridge.BrokerPublicKeyPEM, oidcbridge.BrokerKID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk}})
}

func (s *Server) validateAuthorizeRequest(clientID, redirectURI, responseType, scope string) (*db.Target, string, bool) {
	if responseType != "code" {
		return nil, "only response_type=code is supported", false
	}
	if clientID == "" || redirectURI == "" {
		return nil, "client_id and redirect_uri are required", false
	}
	if scope == "" {
		scope = "openid email profile"
	}
	if !strings.Contains(" "+scope+" ", " openid ") {
		return nil, "scope must include openid", false
	}
	target, err := s.db.GetTargetByClientID(clientID)
	if err != nil || !target.Enabled || target.RedirectURL != redirectURI {
		return nil, "invalid oidc client or redirect_uri", false
	}
	return target, scope, true
}

func (s *Server) emailsForTarget(userID, targetID int64) []db.UserEmail {
	emails, _ := s.db.ListUserEmails(userID)
	var out []db.UserEmail
	for _, e := range emails {
		if !e.Enabled {
			continue
		}
		d, err := s.db.GetDomain(e.DomainID)
		if err == nil && d.TargetID == targetID {
			out = append(out, e)
		}
	}
	return out
}

func (s *Server) handleProviderAuthorize(w http.ResponseWriter, r *http.Request) {
	if _, err := s.currentUser(r); err != nil {
		http.Redirect(w, r, "/oidc/start", http.StatusFound)
		return
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	redirectURI := strings.TrimSpace(r.URL.Query().Get("redirect_uri"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	nonce := strings.TrimSpace(r.URL.Query().Get("nonce"))
	selectedEmailID := parseInt64(r.URL.Query().Get("selected_email_id"))
	target, normalizedScope, ok := s.validateAuthorizeRequest(clientID, redirectURI, r.URL.Query().Get("response_type"), scope)
	if !ok {
		http.Error(w, normalizedScope, http.StatusBadRequest)
		return
	}
	user, _ := s.currentUser(r)
	emails := s.emailsForTarget(user.ID, target.ID)
	if len(emails) == 0 {
		http.Error(w, "no mapped email available for this application", http.StatusBadRequest)
		return
	}
	if selectedEmailID == 0 {
		sess, _ := s.getSession(r)
		if raw, ok := sess.Values["preferred_email:"+clientID].(int64); ok {
			selectedEmailID = raw
		}
		if raw, ok := sess.Values["preferred_email:"+clientID].(int); ok {
			selectedEmailID = int64(raw)
		}
	}
	if selectedEmailID > 0 {
		for _, e := range emails {
			if e.ID == selectedEmailID {
				sess, _ := s.getSession(r)
				delete(sess.Values, "preferred_email:"+clientID)
				_ = sess.Save(r, w)
				s.completeProviderAuthorize(w, r, target, user, e.ID, redirectURI, normalizedScope, nonce, state)
				return
			}
		}
		http.Error(w, "selected email does not belong to this application", http.StatusBadRequest)
		return
	}
	if len(emails) == 1 {
		s.completeProviderAuthorize(w, r, target, user, emails[0].ID, redirectURI, normalizedScope, nonce, state)
		return
	}
	s.renderer.Render(w, "authorize_select.html", AuthorizeSelectData{Title: "Select Email", Target: target, Emails: emails, ClientID: clientID, RedirectURI: redirectURI, Scope: normalizedScope, State: state, Nonce: nonce})
}

func (s *Server) handleProviderAuthorizeSelect(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	clientID := strings.TrimSpace(r.FormValue("client_id"))
	redirectURI := strings.TrimSpace(r.FormValue("redirect_uri"))
	state := strings.TrimSpace(r.FormValue("state"))
	scope := strings.TrimSpace(r.FormValue("scope"))
	nonce := strings.TrimSpace(r.FormValue("nonce"))
	emailID := parseInt64(r.FormValue("email_id"))
	target, normalizedScope, ok := s.validateAuthorizeRequest(clientID, redirectURI, "code", scope)
	if !ok {
		http.Error(w, normalizedScope, http.StatusBadRequest)
		return
	}
	user, _ := s.currentUser(r)
	email, err := s.db.GetUserEmail(emailID, user.ID)
	if err != nil || !email.Enabled {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}
	d, err := s.db.GetDomain(email.DomainID)
	if err != nil || d.TargetID != target.ID {
		http.Error(w, "email does not belong to this application", http.StatusBadRequest)
		return
	}
	s.completeProviderAuthorize(w, r, target, user, email.ID, redirectURI, normalizedScope, nonce, state)
}

func (s *Server) completeProviderAuthorize(w http.ResponseWriter, r *http.Request, target *db.Target, user *db.User, emailID int64, redirectURI, scope, nonce, state string) {
	code := randomCode()
	if err := s.db.CreateBrokerCode(code, target.ID, user.ID, emailID, redirectURI, scope, nonce, state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sep := "?"
	if strings.Contains(redirectURI, "?") {
		sep = "&"
	}
	http.Redirect(w, r, redirectURI+sep+"code="+urlQueryEscape(code)+"&state="+urlQueryEscape(state), http.StatusFound)
}

func resolveClientCredentials(r *http.Request) (string, string) {
	clientID := strings.TrimSpace(r.FormValue("client_id"))
	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "basic ") {
		if decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(auth[6:])); err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				clientID = strings.TrimSpace(parts[0])
				clientSecret = parts[1]
			}
		}
	}
	return clientID, clientSecret
}

func (s *Server) handleProviderToken(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.FormValue("grant_type") != "authorization_code" {
		http.Error(w, "unsupported grant_type", http.StatusBadRequest)
		return
	}
	clientID, clientSecret := resolveClientCredentials(r)
	code := strings.TrimSpace(r.FormValue("code"))
	redirectURI := strings.TrimSpace(r.FormValue("redirect_uri"))
	target, err := s.db.GetTargetByClientID(clientID)
	if err != nil || !target.Enabled || target.ClientSecret != clientSecret {
		http.Error(w, "invalid client credentials", http.StatusUnauthorized)
		return
	}
	brokerCode, err := s.db.GetBrokerCode(code)
	if err != nil || brokerCode.UsedAt != "" || brokerCode.TargetID != target.ID || brokerCode.RedirectURI != redirectURI {
		http.Error(w, "invalid authorization code", http.StatusBadRequest)
		return
	}
	if exp, err := time.Parse(time.RFC3339, brokerCode.ExpiresAt); err == nil && time.Now().UTC().After(exp) {
		http.Error(w, "authorization code expired", http.StatusBadRequest)
		return
	}
	_ = s.db.MarkBrokerCodeUsed(brokerCode.ID)
	user, _ := s.db.GetUserByID(brokerCode.UserID)
	userEmail, _ := s.db.GetUserEmailByID(brokerCode.UserEmailID)
	accessToken := randomCode()
	_ = s.db.CreateBrokerToken(accessToken, target.ID, user.ID, userEmail.ID, brokerCode.Scope)
	idToken, err := oidcbridge.BuildIDToken(issuerBase(s.cfg.BaseURL), target.ClientID, brokerCode.Nonce, brokerSubject(target.ID, userEmail.Email), userEmail.Email, brokerPreferredUsername(userEmail.Email), user.Name, oidcbridge.BrokerPrivateKeyPEM, oidcbridge.BrokerKID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": accessToken, "token_type": "Bearer", "expires_in": 3600, "scope": brokerCode.Scope, "id_token": idToken})
}

func (s *Server) handleProviderUserinfo(w http.ResponseWriter, r *http.Request) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	brokerToken, err := s.db.GetBrokerToken(strings.TrimSpace(auth[7:]))
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	if exp, err := time.Parse(time.RFC3339, brokerToken.ExpiresAt); err == nil && time.Now().UTC().After(exp) {
		http.Error(w, "token expired", http.StatusUnauthorized)
		return
	}
	user, _ := s.db.GetUserByID(brokerToken.UserID)
	userEmail, _ := s.db.GetUserEmailByID(brokerToken.UserEmailID)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"sub": brokerSubject(brokerToken.TargetID, userEmail.Email), "email": userEmail.Email, "email_verified": true, "preferred_username": brokerPreferredUsername(userEmail.Email), "name": user.Name})
}

func brokerSubject(targetID int64, email string) string {
	return "target:" + strconv.FormatInt(targetID, 10) + ":email:" + strings.ToLower(strings.TrimSpace(email))
}

func brokerPreferredUsername(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	if local, _, ok := strings.Cut(email, "@"); ok && local != "" {
		return local
	}
	return email
}
