package handlers

import (
	"log/slog"
	"net/http"

	"wtw/internal/db"
)

type UsersData struct {
	PageData
	Users []db.ListUsersRow
}

func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	pd := basePageData(r, "users")
	if !pd.IsAdmin {
		http.Redirect(w, r, "/recs", http.StatusSeeOther)
		return
	}

	users, err := h.queries.ListUsers(r.Context())
	if err != nil {
		slog.Error("failed to list users", "error", err)
		users = []db.ListUsersRow{}
	}

	data := UsersData{
		PageData: pd,
		Users:    users,
	}

	if isHTMX(r) {
		renderPartial(w, "users-content", data)
		return
	}
	renderPage(w, "users.html", data)
}
