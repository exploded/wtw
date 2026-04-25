package handlers

import (
	"net/http"

	"wtw/internal/db"
	"wtw/internal/middleware"
	"wtw/internal/recommend"
)

type RecsData struct {
	PageData
	Verdict       *recommend.Verdict
	RecsTonight   []ShowWithRating
	BecauseTitle  string
	BecauseShows  []ShowWithRating
	ComfortShows  []ShowWithRating
	HasEnoughData bool
	RatedCount    int
}

func (h *Handler) Recs(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	ratings, _ := h.queries.GetUserRatings(r.Context(), userID)
	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	data := RecsData{
		PageData:   h.basePageDataWithPartners(r, "recs"),
		RatedCount: len(ratings),
	}

	if len(ratings) < 3 {
		data.HasEnoughData = false
		if isHTMX(r) {
			renderPartial(w, "recs-content", data)
			return
		}
		renderPage(w, "recs.html", data)
		return
	}

	data.HasEnoughData = true

	// Get verdict
	verdict, err := h.engine.GetVerdict(r.Context(), userID)
	if err == nil {
		data.Verdict = verdict
	}

	// Tonight's recs
	tonight, _ := h.engine.TopN(r.Context(), userID, 6)
	data.RecsTonight = showsToSWR(tonight, ratingMap)

	// Because you liked...
	title, because, _ := h.engine.BecauseYouLiked(r.Context(), userID)
	data.BecauseTitle = title
	data.BecauseShows = showsToSWR(because, ratingMap)

	// Comfort rewatches
	comfort, _ := h.engine.ComfortRewatches(r.Context(), userID)
	data.ComfortShows = showsToSWR(comfort, ratingMap)

	if isHTMX(r) {
		renderPartial(w, "recs-content", data)
		return
	}
	renderPage(w, "recs.html", data)
}

func (h *Handler) RecsNewVerdict(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	_ = h.queries.DeleteVerdict(r.Context(), userID)
	w.Header().Set("HX-Redirect", "/recs")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RecsRefresh(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	ratings, _ := h.queries.GetUserRatings(r.Context(), userID)
	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	tonight, _ := h.engine.TopN(r.Context(), userID, 6)
	renderPartial(w, "poster-row", showsToSWR(tonight, ratingMap))
}

func showsToSWR(shows []db.Show, ratingMap map[string]string) []ShowWithRating {
	var result []ShowWithRating
	for _, s := range shows {
		result = append(result, ShowWithRating{Show: s, Rating: ratingMap[s.ID]})
	}
	return result
}
