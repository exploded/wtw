package handlers

import (
	"log/slog"
	"net/http"

	"wtw/internal/db"
)

func (h *Handler) Landing(w http.ResponseWriter, r *http.Request) {
	// If user has a valid session, redirect to recs
	cookie, err := r.Cookie("session")
	if err == nil {
		sess, _, err := h.store.GetWithUser(r.Context(), cookie.Value)
		if err == nil && sess != nil {
			http.Redirect(w, r, "/recs", http.StatusSeeOther)
			return
		}
	}

	// Get shows for the decorative poster wall
	shows, err := h.queries.GetShows(r.Context())
	if err != nil {
		slog.Error("failed to get shows for landing", "error", err)
	}

	data := struct {
		Shows []db.Show
	}{
		Shows: shows,
	}

	renderPage(w, "landing.html", data)
}
