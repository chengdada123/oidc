package oidcbridge

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type BrokerClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
}

type BrokerKeypair struct {
	PrivatePEM string
	PublicPEM  string
	KID        string
}

func GenerateBrokerKeypair() (*BrokerKeypair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})
	pubDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	sum := sha256.Sum256(pubDER)
	kid := base64.RawURLEncoding.EncodeToString(sum[:8])
	return &BrokerKeypair{PrivatePEM: string(privPEM), PublicPEM: string(pubPEM), KID: kid}, nil
}

func BuildDiscovery(issuer string) map[string]any {
	issuer = strings.TrimRight(issuer, "/")
	return map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oauth/authorize",
		"token_endpoint":                        issuer + "/oauth/token",
		"userinfo_endpoint":                     issuer + "/oauth/userinfo",
		"jwks_uri":                              issuer + "/oauth/jwks.json",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
		"claims_supported":                      []string{"sub", "name", "email", "email_verified", "given_name", "family_name", "preferred_username"},
	}
}

func BuildJWK(publicPEM, kid string) (map[string]any, error) {
	block, _ := pem.Decode([]byte(publicPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid public key pem")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not rsa")
	}
	return map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}, nil
}

func SignRS256JWT(privatePEM string, payload map[string]any, kid string) (string, error) {
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil {
		return "", fmt.Errorf("invalid private key pem")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	header := map[string]any{"typ": "JWT", "alg": "RS256", "kid": kid}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)
	h64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	p64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := h64 + "." + p64
	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func BuildIDToken(issuer, audience, nonce, sub, email, preferredUsername, name, privatePEM, kid string) (string, error) {
	now := time.Now().Unix()
	payload := map[string]any{
		"iss":                issuer,
		"aud":                audience,
		"sub":                sub,
		"email":              email,
		"email_verified":     true,
		"preferred_username": preferredUsername,
		"name":               name,
		"iat":                now,
		"exp":                now + 3600,
	}
	if nonce != "" {
		payload["nonce"] = nonce
	}
	return SignRS256JWT(privatePEM, payload, kid)
}
