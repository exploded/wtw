# What to Watch — Claude Code Handoff

A clean static UI is in `app.html`, `styles.css`, `data.js`, `app.js`. The HTML uses an htmx-friendly architecture: every screen is a `<section data-screen="<name>">` and every nav element is `data-go="<screen>"`. Below is everything you need to wire it up as a Go / htmx / SQLite / sqlc app hosted at `wtw.mchugh.ai`.

---

## Stack

- **Go** 1.22+ — `net/http` + `chi` router
- **htmx** 1.9.x — single `<script src="https://unpkg.com/htmx.org@1.9.12">` in `<head>`
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGO)
- **sqlc** for typed queries
- **Google OAuth 2.0** for sign-in (golang.org/x/oauth2/google)
- **Sessions**: `gorilla/sessions` with a SQLite-backed store, or signed cookies (`securecookie`)
- **Templates**: Go `html/template` — split per partial so htmx can swap them
- **TMDB API** for poster paths + show metadata (server-side fetch + cache to SQLite)

No build step, no JS framework. Vanilla JS only for the rating-button toggle (already in `app.js`).

---

## Project layout

```
wtw/
├── cmd/wtw/main.go
├── internal/
│   ├── auth/google.go
│   ├── db/queries.sql
│   ├── db/schema.sql
│   ├── db/sqlc.yaml
│   ├── db/gen/         # sqlc output
│   ├── handlers/
│   │   ├── landing.go
│   │   ├── onboarding.go
│   │   ├── catalog.go
│   │   ├── recs.go
│   │   ├── ratings.go
│   │   └── profile.go
│   ├── tmdb/client.go
│   ├── recommend/engine.go
│   └── session/session.go
├── templates/
│   ├── layout.html
│   ├── landing.html
│   ├── onboarding.html
│   ├── catalog.html
│   ├── recs.html
│   ├── profile.html
│   └── partials/
│       ├── poster-card.html
│       ├── poster-grid.html
│       └── poster-row.html
├── static/
│   ├── styles.css       # copy from this project
│   ├── app.js           # copy + trim — see "JS surface" below
│   └── favicon.ico
├── go.mod
└── README.md
```

---

## SQLite schema (`internal/db/schema.sql`)

```sql
CREATE TABLE users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  google_sub    TEXT NOT NULL UNIQUE,
  email         TEXT NOT NULL,
  name          TEXT NOT NULL,
  avatar_url    TEXT,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE shows (
  id            TEXT PRIMARY KEY,           -- e.g. "the-leftovers"
  tmdb_id       INTEGER UNIQUE,
  title         TEXT NOT NULL,
  year          TEXT NOT NULL,
  poster_path   TEXT,                       -- TMDB path, e.g. /jdZecZ...jpg
  genre         TEXT,
  synopsis      TEXT,
  popularity    REAL DEFAULT 0,
  cached_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ratings (
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  show_id       TEXT    NOT NULL REFERENCES shows(id),
  rating        TEXT    NOT NULL CHECK (rating IN ('liked','disliked','unseen')),
  rated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, show_id)
);

CREATE TABLE sessions (
  token         TEXT PRIMARY KEY,
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at    DATETIME NOT NULL
);

CREATE INDEX idx_ratings_user ON ratings(user_id);
```

Seed the `shows` table from `data.js` on first boot — port the array to a Go slice and `INSERT OR IGNORE`.

---

## sqlc queries (`internal/db/queries.sql`)

```sql
-- name: UpsertUser :one
INSERT INTO users (google_sub, email, name, avatar_url)
VALUES (?, ?, ?, ?)
ON CONFLICT(google_sub) DO UPDATE SET email=excluded.email, name=excluded.name, avatar_url=excluded.avatar_url
RETURNING *;

-- name: GetShows :many
SELECT * FROM shows ORDER BY popularity DESC;

-- name: GetUserRatings :many
SELECT show_id, rating FROM ratings WHERE user_id = ?;

-- name: SetRating :exec
INSERT INTO ratings (user_id, show_id, rating)
VALUES (?, ?, ?)
ON CONFLICT(user_id, show_id) DO UPDATE SET rating = excluded.rating, rated_at = CURRENT_TIMESTAMP;

-- name: DeleteRating :exec
DELETE FROM ratings WHERE user_id = ? AND show_id = ?;

-- name: CountRatings :one
SELECT
  COUNT(*) FILTER (WHERE rating='liked')    AS liked,
  COUNT(*) FILTER (WHERE rating='disliked') AS disliked,
  COUNT(*) FILTER (WHERE rating='unseen')   AS unseen,
  COUNT(*)                                  AS total
FROM ratings WHERE user_id = ?;
```

---

## Routes

| Method | Path                          | Returns                              | Notes |
|--------|-------------------------------|--------------------------------------|-------|
| GET    | `/`                           | landing.html (or redirect to /recs)  | Public |
| GET    | `/auth/google`                | redirect to Google OAuth             | |
| GET    | `/auth/google/callback`       | redirect to /onboarding or /recs     | Sets session cookie |
| POST   | `/auth/logout`                | redirect to /                        | |
| GET    | `/onboarding`                 | onboarding.html                      | Auth required |
| GET    | `/catalog`                    | catalog.html                         | Auth required |
| GET    | `/recs`                       | recs.html                            | Auth required |
| GET    | `/recs/refresh`               | partials/poster-row.html (HX-only)   | Returns 6 fresh posters |
| GET    | `/profile`                    | profile.html                         | Auth required |
| POST   | `/ratings/{show_id}`          | partials/poster-card.html            | form: `rating=liked\|disliked\|unseen` |
| DELETE | `/ratings/{show_id}`          | partials/poster-card.html            | clears rating |
| GET    | `/catalog/filter`             | partials/poster-grid.html            | query: `?filter=liked&q=...` |

All authenticated routes check the session cookie via middleware; redirect to `/` if missing.

---

## htmx wiring

The current static UI uses `data-go` for client-side screen swaps. Replace with hx attributes:

```html
<!-- before (static) -->
<button data-go="recs">For you</button>

<!-- after (htmx) -->
<a href="/recs" hx-get="/recs" hx-target="#app" hx-push-url="true">For you</a>
```

Rating button (in `partials/poster-card.html`):

```html
<button class="rate-btn rate-btn--liked {{if eq .Rating "liked"}}is-active{{end}}"
        hx-post="/ratings/{{.ID}}"
        hx-vals='{"rating":"liked"}'
        hx-target="closest .poster-card"
        hx-swap="outerHTML">
  <svg>...</svg>
</button>
```

The server returns a freshly-rendered `<div class="poster-card" data-rating="liked" ...>` — same markup the static `app.js` produces, just rendered server-side.

For the **catalog filters/search**, use `hx-get` with `hx-trigger="input changed delay:200ms"` on the search input and `hx-trigger="click"` on filter buttons, both targeting `#catalogGrid`.

For the **refresh recs** button, target `#recsRow` with `hx-get="/recs/refresh"`.

---

## Recommendation engine (`internal/recommend/engine.go`)

Start simple. Real ML can come later.

1. Compute a genre/keyword vector from the user's `liked` shows (TMDB has `genre_ids` + `keywords` per show).
2. Score every unrated show by cosine similarity to that vector.
3. Penalize anything sharing genre/keywords with `disliked` shows.
4. Return the top N as `RECS_TONIGHT`. For the "Because you liked X" rows, pick the user's top liked show and run a TMDB `/tv/{id}/recommendations` query, filtered by what they haven't already disliked.

For the AI verdict copy on `/recs`, call Claude (or any LLM) with a prompt like:

> You are an opinionated cinephile. The user liked: {liked_titles}. They disliked: {disliked_titles}. Recommend ONE TV series from this candidate list and write 2 short sentences explaining why, in the voice of a knowledgeable film-friend. Candidates: {top_5_by_score}.

Cache the verdict for ~24h per user — recomputing on every `/recs` is wasteful.

---

## Google OAuth (`internal/auth/google.go`)

Standard `golang.org/x/oauth2/google`. Required env vars:

```
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URL=https://wtw.mchugh.ai/auth/google/callback
SESSION_KEY=<32-byte hex>
TMDB_API_KEY=...
DATABASE_URL=file:wtw.db?_fk=1
```

Scopes: `openid email profile`. On callback, fetch `https://www.googleapis.com/oauth2/v3/userinfo`, upsert into `users`, mint a session token, store in `sessions`, set as HTTP-only cookie.

---

## TMDB

Use `https://api.themoviedb.org/3/tv/{id}` to fetch metadata; build poster URLs as `https://image.tmdb.org/t/p/w500{poster_path}`. The `data.js` in this project already has 50 real shows with valid `poster_path` values — port them to the seed.

Cache responses in the `shows` table with `cached_at`; refresh older than 30 days.

---

## JS surface (keep tiny)

After moving to htmx, `app.js` shrinks to almost nothing. Keep only:

- The screen-routing was for the static prototype — **delete it**, htmx handles navigation now.
- The rating-button click handler — **delete it**, the button is now `hx-post`.
- The catalog filter wiring — **delete it**, filters are `hx-get`.

What's left: nothing. Optionally a 5-line script that adds a loading shimmer class on `htmx:beforeRequest`.

---

## Deploy

- Build: `CGO_ENABLED=0 go build -o wtw ./cmd/wtw`
- Run behind Caddy (auto-TLS for `wtw.mchugh.ai`):
  ```
  wtw.mchugh.ai {
    reverse_proxy localhost:8080
  }
  ```
- `wtw.db` lives next to the binary; back it up nightly with `litestream` to S3.

---

## Migration order for Claude Code

1. `go mod init`, install deps, scaffold `cmd/wtw/main.go` with chi + a single `/` route returning the static landing page from `templates/landing.html`.
2. Set up SQLite + sqlc; seed `shows` from the array in `data.js`.
3. Implement Google OAuth and session middleware.
4. Convert `app.html` into `templates/*.html`. Each `<section data-screen>` becomes its own template extending `layout.html`.
5. Convert `data-go` clicks to `hx-get`. Convert rating buttons to `hx-post`. Convert filters to `hx-get`. Delete the corresponding JS.
6. Implement the recommendation engine — start with the simple genre-vector scorer; add the LLM verdict last.
7. Wire up TMDB metadata refresh (cron-style goroutine).
8. Deploy behind Caddy.

The visual design system (`styles.css`) does not need changes. Use it as-is.
