package server

import "oidc-bridge/internal/db"

type HomeData struct {
	Title     string
	OIDCReady bool
}

type DashboardEmailRow struct {
	ID         int64
	Email      string
	Note       string
	Enabled    bool
	Domain     string
	TargetName string
}

type AdminDomainRow struct {
	ID          int64
	Domain      string
	Description string
	Enabled     bool
	TargetID    int64
	TargetName  string
}

type AdminUserEmailRow struct {
	ID           int64
	UserID       int64
	UserSub      string
	UserEmail    string
	UserName     string
	UserDisabled bool
	Email        string
	LocalPart    string
	Note         string
	Domain       string
	TargetName   string
	Enabled      bool
}

type DashboardData struct {
	Title      string
	User       *db.User
	Domains    []db.Domain
	Emails     []DashboardEmailRow
	EmailLimit int
	Error      string
}

type AdminLoginData struct {
	Title string
	Error string
}

type AdminPageData struct {
	Title          string
	BaseURL        string
	Domains        []AdminDomainRow
	Targets        []db.Target
	UserEmails     []AdminUserEmailRow
	EmailLimit     int
	UserEmailQuery string
	UserEmailPage  int
	UserEmailPages int
	UserEmailTotal int
	UserEmailPrev  int
	UserEmailNext  int
	Error          string
}

type AuthorizeSelectData struct {
	Title       string
	Target      *db.Target
	Emails      []db.UserEmail
	ClientID    string
	RedirectURI string
	Scope       string
	State       string
	Nonce       string
}
