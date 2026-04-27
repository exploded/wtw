package handlers

import (
	"net/http"
	"strings"

	"wtw/internal/db"
	"wtw/internal/middleware"
)

type UsersData struct {
	PageData
	Users []db.ListUsersRow
}

func (h *Handler) Users(w http.ResponseWriter, r *http.Request) {
	email := middleware.GetUserEmail(r.Context())
	if !strings.EqualFold(email, adminEmail) {
		http.Redirect(w, r, "/recs", http.StatusSeeOther)
		return
	}

	users, _ := h.queries.ListUsers(r.Context())

	data := UsersData{
		PageData: h.basePageDataWithPartners(r, "users"),
		Users:    users,
	}

	if isHTMX(r) {
		renderPartial(w, "users-content", data)
		return
	}
	renderPage(w, "users.html", data)
}
