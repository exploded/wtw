package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.windowStart) > rl.window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
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

func rateLimitMiddleware(limiter *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
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

	// Seed shows
	if err := db.SeedAllShows(database); err != nil {
		slog.Error("failed to seed shows", "error", err)
		os.Exit(1)
	}

	queries := db.New(database)

	// Session store
	store := session.NewStore(queries)
	go store.CleanupLoop()

	// Auth handler
	authHandler := auth.NewHandler(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
		queries,
		store,
	)

	// TMDB client — refresh poster paths in background
	tmdbClient := tmdb.NewClient(os.Getenv("TMDB_API_KEY"), queries)
	tmdbClient.StartRefreshLoop(context.Background())

	// Recommendation engine
	engine := recommend.NewEngine(queries, os.Getenv("ANTHROPIC_API_KEY"))

	// Handlers
	h := handlers.New(queries, store, engine)

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
	mux.HandleFunc("GET /recs", middleware.RequireAuth(store, h.Recs))
	mux.HandleFunc("GET /recs/refresh", middleware.RequireAuth(store, h.RecsRefresh))
	mux.HandleFunc("GET /profile", middleware.RequireAuth(store, h.Profile))
	mux.HandleFunc("POST /ratings/{show_id}", middleware.RequireAuth(store, h.SetRating))
	mux.HandleFunc("DELETE /ratings/{show_id}", middleware.RequireAuth(store, h.DeleteRating))

	// Static files
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheStaticAssets(fileServer)))

	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})

	// Middleware chain
	limiter := newRateLimiter(60, time.Minute)
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

	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server exited")
}

func cacheStaticAssets(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		next.ServeHTTP(w, r)
	})
}
