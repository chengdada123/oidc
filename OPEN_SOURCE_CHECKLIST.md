# Release Notes

## Before Publishing

- review README examples and public URLs
- confirm `.env.example` contains no private values
- verify `.gitignore` excludes local databases and binaries
- ensure `docker-compose.yml` and `Dockerfile` build cleanly
- remove local test-only artifacts from the repo root if present

## Recommended Follow-up Work

- persist JWKS signing keys across restarts
- add integration tests for authorize/token/userinfo
- add health endpoint and structured logs
- document reverse proxy deployment examples
- add optional Postgres support if needed later
