package oidcbridge

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"oidc-bridge/internal/config"
)

type Provider struct {
	Config       *config.Config
	OIDCProvider *oidc.Provider
	Verifier     *oidc.IDTokenVerifier
	OAuth2Config *oauth2.Config
}

type UserClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewProvider(ctx context.Context, cfg *config.Config) (*Provider, error) {
	provider, err := oidc.NewProvider(ctx, cfg.VPS8Issuer)
	if err != nil {
		return nil, fmt.Errorf("new oidc provider: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.VPS8ClientID,
		ClientSecret: cfg.VPS8ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.BaseURL + "/oidc/callback",
		Scopes:       cfg.VPS8Scopes,
	}

	return &Provider{
		Config:       cfg,
		OIDCProvider: provider,
		Verifier: provider.Verifier(&oidc.Config{
			ClientID: cfg.VPS8ClientID,
		}),
		OAuth2Config: oauth2Config,
	}, nil
}

func (p *Provider) AuthCodeURL(state string) string {
	return p.OAuth2Config.AuthCodeURL(state)
}

func (p *Provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.OAuth2Config.Exchange(ctx, code)
}

func (p *Provider) VerifyAndFetchUser(ctx context.Context, token *oauth2.Token) (*UserClaims, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("missing id_token")
	}

	idToken, err := p.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify id_token: %w", err)
	}

	var claims UserClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse id_token claims: %w", err)
	}

	if claims.Email == "" {
		userInfo, err := p.OIDCProvider.UserInfo(ctx, oauth2.StaticTokenSource(token))
		if err == nil {
			_ = userInfo.Claims(&claims)
		}
	}

	if claims.Sub == "" || claims.Email == "" {
		return nil, fmt.Errorf("missing required oidc claims")
	}

	return &claims, nil
}

func NewState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GetContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}
