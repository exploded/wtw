package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"wtw/internal/auth"
	"wtw/internal/db"
	"wtw/internal/handlers"
	"wtw/internal/middleware"
	"wtw/internal/recommend"
	"wtw/internal/session"
	"wtw/internal/tmdb"
)

// rateLimiter tracks request counts per IP using a sliding window.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	window   time.Duration
}

type visitor struct {
	count       int
	windowStart time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
}

func (rl *rateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.windowStart) > rl.window {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists || now.Sub(v.windowStart) > rl.window {
		rl.visitors[ip] = &visitor{count: 1, windowStart: now}
		return true
	}

	v.count++
	return v.count <= rl.rate
}

// clientIP extracts the client IP, preferring X-Forwarded-For behind a proxy.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First entry is the original client
		if ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func rateLimitMiddleware(limiter *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !limiter.allow(ip) {
			slog.Warn("rate limit exceeded", "ip", ip)
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	isProd := os.Getenv("PROD") == "True"

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	port = ":" + port

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("starting wtw", "prod", isProd, "port", port)

	// Open SQLite database
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "file:wtw.db?_fk=1"
	}
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	database.SetMaxOpenConns(1)
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		slog.Error("failed to set WAL mode", "error", err)
		os.Exit(1)
	}
	if _, err := database.Exec("PRAGMA foreign_keys=ON"); err != nil {
		slog.Error("failed to enable foreign keys", "error", err)
		os.Exit(1)
	}

	// Initialize schema
	if _, err := database.Exec(db.SchemaSQL); err != nil {
		slog.Error("failed to initialize schema", "error", err)
		os.Exit(1)
	}

	// Migrate ratings CHECK constraint to include 'favourite'
	if err := migrateRatingsConstraint(database); err != nil {
		slog.Error("failed to migrate ratings constraint", "error", err)
		os.Exit(1)
	}

	// Seed shows
	if err := db.SeedAllShows(database); err != nil {
		slog.Error("failed to seed shows", "error", err)
		os.Exit(1)
	}

	queries := db.New(database)

	// Cancellable context for background goroutines
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()

	// Session store
	store := session.NewStore(queries)
	go store.CleanupLoop(bgCtx)

	// Auth handler
	authHandler := auth.NewHandler(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		queries,
		store,
		isProd,
	)

	// TMDB client -- refresh poster paths in background
	tmdbClient := tmdb.NewClient(os.Getenv("TMDB_API_KEY"), queries)
	tmdbClient.StartRefreshLoop(bgCtx)

	// Recommendation engine
	engine := recommend.NewEngine(queries, os.Getenv("ANTHROPIC_API_KEY"), tmdbClient)

	// Handlers
	h := handlers.New(queries, store, engine, tmdbClient)

	// Rate limiter with cancellable cleanup
	limiter := newRateLimiter(60, time.Minute)
	go limiter.cleanup(bgCtx)

	// Routes
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("GET /{$}", h.Landing)
	mux.HandleFunc("GET /auth/google", authHandler.Login)
	mux.HandleFunc("GET /auth/google/callback", authHandler.Callback)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)

	// Protected
	mux.HandleFunc("GET /onboarding", middleware.RequireAuth(store, h.Onboarding))
	mux.HandleFunc("GET /catalog", middleware.RequireAuth(store, h.Catalog))
	mux.HandleFunc("GET /catalog/filter", middleware.RequireAuth(store, h.CatalogFilter))
	mux.HandleFunc("GET /catalog/tmdb-search", middleware.RequireAuth(store, h.CatalogSearchTMDB))
	mux.HandleFunc("POST /catalog/add-tmdb/{tmdb_id}", middleware.RequireAuth(store, h.CatalogAddTMDB))
	mux.HandleFunc("GET /recs", middleware.RequireAuth(store, h.Recs))
	mux.HandleFunc("GET /recs/refresh", middleware.RequireAuth(store, h.RecsRefresh))
	mux.HandleFunc("POST /recs/new-verdict", middleware.RequireAuth(store, h.RecsNewVerdict))
	mux.HandleFunc("GET /profile", middleware.RequireAuth(store, h.Profile))
	mux.HandleFunc("GET /users", middleware.RequireAuth(store, h.Users))
	mux.HandleFunc("POST /ratings/{show_id}", middleware.RequireAuth(store, h.SetRating))
	mux.HandleFunc("DELETE /ratings/{show_id}", middleware.RequireAuth(store, h.DeleteRating))

	// Partner / Together
	mux.HandleFunc("GET /together/{id}", middleware.RequireAuth(store, h.Together))
	mux.HandleFunc("GET /together/{id}/refresh", middleware.RequireAuth(store, h.TogetherRefresh))
	mux.HandleFunc("POST /partners/add", middleware.RequireAuth(store, h.AddPartner))
	mux.HandleFunc("POST /partners/{id}/accept", middleware.RequireAuth(store, h.AcceptPartner))
	mux.HandleFunc("POST /partners/{id}/remove", middleware.RequireAuth(store, h.RemovePartner))

	// Static files
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheStaticAssets(fileServer)))

	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})

	// Middleware chain
	handler := middleware.RequestLogger(
		rateLimitMiddleware(limiter,
			middleware.SecurityHeaders(isProd, mux),
		),
	)

	srv := &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Use a channel to propagate server errors to main goroutine
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down server...")
	case err := <-serverErr:
		slog.Error("server error", "error", err)
	}

	// Cancel background goroutines before shutting down the server
	bgCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server exited")
}

// migrateRatingsConstraint recreates the ratings table if the CHECK constraint
// does not include 'favourite'. Needed because CREATE TABLE IF NOT EXISTS skips
// existing tables, so the schema update alone cannot fix old databases.
func migrateRatingsConstraint(database *sql.DB) error {
	var tableSql string
	err := database.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='ratings'").Scan(&tableSql)
	if err != nil {
		return nil // table doesn't exist yet, schema will create it
	}
	if strings.Contains(tableSql, "'favourite'") {
		return nil // already migrated
	}
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
		CREATE TABLE ratings_new (
			user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			show_id   TEXT    NOT NULL REFERENCES shows(id),
			rating    TEXT    NOT NULL CHECK (rating IN ('favourite','liked','disliked','unseen')),
			rated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, show_id)
		);
		INSERT INTO ratings_new SELECT * FROM ratings;
		DROP TABLE ratings;
		ALTER TABLE ratings_new RENAME TO ratings;
		CREATE INDEX IF NOT EXISTS idx_ratings_user ON ratings(user_id);
	`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func cacheStaticAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		next.ServeHTTP(w, r)
	})
}
