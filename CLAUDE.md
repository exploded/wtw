# wtw — "What to Watch" TV recommender

Rate TV series you've loved/hated, then an AI cinephile (Claude) writes tonight's
verdict, "because you liked…" rows, and a taste profile; "together mode" links two
accounts and finds the overlap. Google sign-in, TMDB catalog/search.
Live at https://wtw.mchugh.au.

After any UI change, verify it in the browser (use the chrome-devtools MCP tools if
available) before reporting done. Do not claim a fix works without exercising the flow.

## Skills
Use `go-htmx-skill` for template/HTMX/UI work and `sqlc-sqlite` for query work.

## Stack
Go 1.26, stdlib `net/http`, html/template + HTMX, modernc.org/sqlite (pure Go) via
sqlc, Google OAuth (`golang.org/x/oauth2`), TMDB API, Anthropic Messages API
(model `claude-sonnet-4-20250514` in `internal/recommend/engine.go`).

Logs ship to the monitor portal via `github.com/exploded/monitor/pkg/logship`,
gated on `MONITOR_URL` + `MONITOR_API_KEY` (wired in `cmd/wtw/main.go`). WARN and
above travel; only ERROR raises an alert, so reserve `slog.Error` for things that
actually need attention.

## Build / run (Windows)
```
build.bat        # go build -o wtw.exe ./cmd/wtw/, creates .env from example, runs it
run.bat          # runs existing wtw.exe with .env loaded
go build ./...
```
No test files exist. Env (`.env.example`): GOOGLE_CLIENT_ID/SECRET/REDIRECT_URL,
SESSION_KEY, ANTHROPIC_API_KEY, TMDB_API_KEY, PORT (8991), PROD, MONITOR_*.

## sqlc
`sqlc generate` — reads `internal/db/schema.sql` + `internal/db/queries.sql` →
regenerates `internal/db/`. schema.sql is `go:embed`-ed and applied at startup.

## Layout
```
cmd/wtw/                  entry
internal/auth/            Google OAuth flow
internal/db/              sqlc output + seed.go (catalog seed) + slug.go
internal/handlers/        landing, onboarding, catalog, ratings, recs, profile,
                          together, users (admin), render.go (template helpers)
internal/middleware/      auth, logging, security headers
internal/recommend/       scoring engine + Anthropic calls
internal/session/         session store
internal/tmdb/            TMDB client
templates/ static/        loaded from disk (NOT embedded; ship with the bundle)
```

## Deployment
- Repo: `exploded/wtw`, default branch `master`. Push to `master` → Actions
  "Test and Deploy" (`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`).
- Bundle = binary + `templates/` + `static/` + `scripts/deploy-wtw`.
- Prod: systemd unit `wtw`, port **8991**, `/var/www/wtw`, Caddy at
  https://wtw.mchugh.au. Provisioning: `scripts/server-setup.sh`.

## Gotchas (verified in code / git history)
- **iOS Safari is the fragile browser here**: the poster rating bar broke on it
  repeatedly (3556e97, dad91af, 6058c7b) — hover media queries and low-specificity
  mobile overrides don't behave; use `max-width` breakpoints and expect to bump
  specificity. Always check mobile Safari after CSS changes near ratings/posters.
- The recommend engine **degrades gracefully without ANTHROPIC_API_KEY** (checked
  in `engine.go`) — verdict/LLM features are skipped, scoring still works. Don't
  hard-fail on a missing key.
- Admin Users page is restricted to `james67@gmail.com` (3c2dacf).
- CSP blocked Cloudflare Insights once already (9e63701) — new external scripts
  need a CSP update in `internal/middleware/security.go`.
- `.local/` is gitignored scratch space (2d4841f) — never commit it.
- Never commit `.env` or `*.db*`.
