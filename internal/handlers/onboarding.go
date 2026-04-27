package handlers

import (
	"log/slog"
	"net/http"

	"wtw/internal/middleware"
)

type OnboardingData struct {
	PageData
	Shows      []ShowWithRating
	RatedCount int
	TotalShows int
}

func (h *Handler) Onboarding(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shows, err := h.queries.GetTopShows(r.Context(), 20)
	if err != nil {
		slog.Error("failed to get top shows", "error", err)
	}
	ratings, err := h.queries.GetUserRatings(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get user ratings", "error", err)
	}

	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	var showsWithRatings []ShowWithRating
	for _, show := range shows {
		showsWithRatings = append(showsWithRatings, ShowWithRating{
			Show:   show,
			Rating: ratingMap[show.ID],
		})
	}

	data := OnboardingData{
		PageData:   basePageData(r, ""),
		Shows:      showsWithRatings,
		RatedCount: len(ratings),
		TotalShows: len(shows),
	}

	if isHTMX(r) {
		renderPartial(w, "onboarding-content", data)
		return
	}
	renderPage(w, "onboarding.html", data)
}
