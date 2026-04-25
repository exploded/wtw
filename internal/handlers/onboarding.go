package handlers

import (
	"net/http"

	"wtw/internal/middleware"
)

type OnboardingData struct {
	PageData
	Shows      []ShowWithRating
	RatedCount int
}

func (h *Handler) Onboarding(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shows, _ := h.queries.GetShows(r.Context())
	ratings, _ := h.queries.GetUserRatings(r.Context(), userID)

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
	}

	if isHTMX(r) {
		renderPartial(w, "onboarding-content", data)
		return
	}
	renderPage(w, "onboarding.html", data)
}
