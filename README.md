# OIDC Bridge

OIDC Bridge is a standalone Go + SQLite service that sits between an upstream OIDC identity source and downstream standard OIDC clients.

It solves a specific problem: one real upstream account needs to present different mapped email identities to different downstream applications.

## What It Does

- Uses an upstream standard OIDC provider for real user authentication
- Provides a user console where users create and manage mapped domain-email identities
- Provides an admin backend for application and domain management
- Exposes a downstream standard OIDC provider for target applications

## Core Model

- One application can bind multiple domains
- One domain can bind exactly one application
- Domains exist to create mapped email identities
- User login flow is: upstream OIDC login -> user console -> select mapped email -> downstream app login
- Email identities are globally unique

## Example Flow

1. A user signs in to OIDC Bridge using the upstream OIDC provider.
2. The user creates `bbs@vps8.bond` in the user console.
3. The `vps8.bond` domain is bound to the `bbs` application.
4. The user clicks `Use this email to sign in`.
5. OIDC Bridge sends the user to the target application's OIDC login entrypoint.
6. The target application starts a normal OIDC authorization request against OIDC Bridge.
7. OIDC Bridge returns claims for the selected mapped email identity.

## Features

- Upstream OIDC login for end users
- Admin password login for backend management
- Auto-generated downstream `client_id` and `client_secret`
- Domain-to-application binding
- Application secret reset
- Application edit/delete
- Domain rebind/edit/delete
- Standard OIDC discovery, authorization, token, JWKS, and userinfo endpoints
- Mobile-friendly admin and user console

## OIDC Provider Endpoints

Assuming `BASE_URL=https://bridge.example.com`:

- Discovery: `https://bridge.example.com/.well-known/openid-configuration`
- Authorize: `https://bridge.example.com/oauth/authorize`
- Token: `https://bridge.example.com/oauth/token`
- Userinfo: `https://bridge.example.com/oauth/userinfo`
- JWKS: `https://bridge.example.com/oauth/jwks.json`

Supported scopes:

- `openid`
- `profile`
- `email`

Supported token auth methods:

- `client_secret_post`
- `client_secret_basic`

## Claims

OIDC Bridge returns mapped identity claims for the selected email, including:

- `sub`
- `email`
- `email_verified`
- `preferred_username`
- `name`

## Configuration

Copy `.env.example` to `.env` and adjust values.

Important variables:

- `PORT`
- `BASE_URL`
- `SESSION_SECRET`
- `ADMIN_PASSWORD`
- `DB_PATH`
- `VPS8_OIDC_ISSUER`
- `VPS8_OIDC_CLIENT_ID`
- `VPS8_OIDC_CLIENT_SECRET`
- `VPS8_OIDC_SCOPES`

## Local Run

```bash
go build -o oidc-bridge ./cmd/bridge
./oidc-bridge
```

### Helper Scripts

Windows PowerShell:

```powershell
./scripts/run.ps1 -Port 8080 -StopFirst
./scripts/stop.ps1 -Port 8080
```

Linux / Ubuntu / Debian / CentOS / Alpine:

```sh
chmod +x ./scripts/run.sh ./scripts/stop.sh
STOP_FIRST=1 PORT=8080 ./scripts/run.sh
PORT=8080 ./scripts/stop.sh
```

## Docker

```bash
docker compose up -d --build
```

The default compose setup:

- builds the service locally
- maps `8080:8080`
- mounts `./data` for SQLite persistence
- reads configuration from `.env`

### Minimal Delivery Goal

A deployer should be able to:

1. copy `.env.example` to `.env`
2. fill `BASE_URL`, `SESSION_SECRET`, `ADMIN_PASSWORD`
3. fill upstream VPS8 OIDC values:
   - `VPS8_OIDC_ISSUER`
   - `VPS8_OIDC_CLIENT_ID`
   - `VPS8_OIDC_CLIENT_SECRET`
4. run `docker compose up -d --build`
5. log into `/admin`
6. use VPS8 account login in the user console

## Admin Setup

In the admin backend you create downstream applications with:

- application name
- login URL
- redirect URL
- enabled flag

The system auto-generates:

- `client_id`
- `client_secret`

### Application Fields

- `login_url`: where the user should be sent to start the target app's OIDC login flow
- `redirect_url`: the target app's registered OIDC callback URL

These two fields are intentionally separate. The login entrypoint is not the callback.

## Project Layout

- `cmd/bridge` - main entrypoint
- `internal/config` - environment config
- `internal/db` - SQLite schema and store layer
- `internal/oidc` - upstream/downstream OIDC logic
- `internal/server` - HTTP handlers and routing
- `internal/web/templates` - HTML templates

## Production Notes

- Use a strong `SESSION_SECRET`
- Put the service behind HTTPS
- Set `BASE_URL` to the public HTTPS origin
- Persist `data/bridge.db`
- Review cookie `Secure` settings before public deployment
- Current JWKS signing key is generated at runtime; persistent key management should be used for long-lived production deployments

## Status

This project is usable, but still young. The main architecture is in place and the downstream provider flow works, but production hardening should continue around:

- persistent signing keys
- broader OIDC interoperability testing
- deployment defaults
- audit/logging and operational polish
- CI coverage for container startup and admin login

## License

MIT

## Security

See [SECURITY.md](./SECURITY.md).
