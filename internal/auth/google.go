package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"wtw/internal/db"
	"wtw/internal/session"
)

type Handler struct {
	config  *oauth2.Config
	queries *db.Queries
	store   *session.Store
	isProd  bool
}

func NewHandler(clientID, clientSecret, redirectURL string, queries *db.Queries, store *session.Store, isProd bool) *Handler {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &Handler{config: cfg, queries: queries, store: store, isProd: isProd}
}

type googleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	state := hex.EncodeToString(b)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.isProd,
	})

	url := h.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	// Validate OAuth state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	token, err := h.config.Exchange(context.Background(), code)
	if err != nil {
		slog.Error("oauth exchange failed", "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}

	client := h.config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		slog.Error("failed to fetch userinfo", "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		slog.Error("failed to decode userinfo", "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}

	// Upsert user
	user, err := h.queries.UpsertUser(r.Context(), db.UpsertUserParams{
		GoogleSub: info.Sub,
		Email:     info.Email,
		Name:      info.Name,
		AvatarUrl: sql.NullString{String: info.Picture, Valid: info.Picture != ""},
	})
	if err != nil {
		slog.Error("failed to upsert user", "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionToken, err := h.store.Create(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to create session", "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   session.MaxAgeSec,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.isProd,
	})

	// Check if user has ratings to decide redirect
	ratings, err := h.queries.GetUserRatings(r.Context(), user.ID)
	if err != nil {
		slog.Error("failed to check user ratings", "error", err)
		http.Redirect(w, r, "/onboarding", http.StatusSeeOther)
		return
	}
	if len(ratings) == 0 {
		http.Redirect(w, r, "/onboarding", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/recs", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		_ = h.store.Delete(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Handle htmx request
	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
