package handlers

import (
	"net/http"

	"wtw/internal/db"
	"wtw/internal/middleware"
)

type ShowWithRating struct {
	Show   db.Show
	Rating string
}

func (h *Handler) SetRating(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	showID := r.PathValue("show_id")
	rating := r.FormValue("rating")

	if rating != "favourite" && rating != "liked" && rating != "disliked" && rating != "unseen" {
		http.Error(w, "invalid rating", http.StatusBadRequest)
		return
	}

	// Toggle: if current rating matches, remove it
	currentRatings, _ := h.queries.GetUserRatings(r.Context(), userID)
	for _, cr := range currentRatings {
		if cr.ShowID == showID && cr.Rating == rating {
			_ = h.queries.DeleteRating(r.Context(), db.DeleteRatingParams{
				UserID: userID,
				ShowID: showID,
			})
			show, _ := h.queries.GetShowByID(r.Context(), showID)
			renderPartial(w, "poster-card", ShowWithRating{Show: show, Rating: ""})
			return
		}
	}

	_ = h.queries.SetRating(r.Context(), db.SetRatingParams{
		UserID: userID,
		ShowID: showID,
		Rating: rating,
	})

	show, _ := h.queries.GetShowByID(r.Context(), showID)
	renderPartial(w, "poster-card", ShowWithRating{Show: show, Rating: rating})
}

func (h *Handler) DeleteRating(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	showID := r.PathValue("show_id")

	_ = h.queries.DeleteRating(r.Context(), db.DeleteRatingParams{
		UserID: userID,
		ShowID: showID,
	})

	show, _ := h.queries.GetShowByID(r.Context(), showID)
	renderPartial(w, "poster-card", ShowWithRating{Show: show, Rating: ""})
}
