package handlers

import (
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
	shows, _ := h.queries.GetShows(r.Context())

	data := struct {
		Shows []db.Show
	}{
		Shows: shows,
	}

	renderPage(w, "landing.html", data)
}
