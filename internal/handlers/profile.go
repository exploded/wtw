package handlers

import (
	"net/http"

	"wtw/internal/db"
	"wtw/internal/middleware"
)

type ProfileData struct {
	PageData
	Liked       int64
	Disliked    int64
	Total       int64
	LikedShows  []db.Show
	TopShows    []db.Show
	ShowCount   int
	MemberSince string
}


func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	counts, _ := h.queries.CountRatings(r.Context(), userID)
	liked := int64(counts.Liked.Float64)
	disliked := int64(counts.Disliked.Float64)

	likedShows, _ := h.queries.GetLikedShows(r.Context(), userID)
	shows, _ := h.queries.GetShows(r.Context())

	// Top shows (up to 5 liked)
	topShows := likedShows
	if len(topShows) > 5 {
		topShows = topShows[:5]
	}

	user, _ := h.queries.GetUserByID(r.Context(), userID)

	data := ProfileData{
		PageData:    basePageData(r, "profile"),
		Liked:       liked,
		Disliked:    disliked,
		Total:       counts.Total,
		LikedShows:  likedShows,
		TopShows:    topShows,
		ShowCount:   len(shows),
		MemberSince: user.CreatedAt.Format("January 2006"),
	}

	if isHTMX(r) {
		renderPartial(w, "profile-content", data)
		return
	}
	renderPage(w, "profile.html", data)
}
