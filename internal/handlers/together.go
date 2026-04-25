package handlers

import (
	"net/http"
	"strconv"

	"wtw/internal/db"
	"wtw/internal/middleware"
	"wtw/internal/recommend"
)

type TogetherData struct {
	PageData
	PartnershipID int64
	PartnerName   string
	PartnerInitials string
	Verdict       *recommend.Verdict
	RecsTonight   []ShowWithRating
	BecauseTitle  string
	BecauseShows  []ShowWithRating
	OverlapShows  []db.Show
	HasEnoughData bool
	MyRatedCount  int
	TheirRatedCount int
}

func (h *Handler) Together(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")
	partnershipID, _ := strconv.ParseInt(idStr, 10, 64)

	partnership, err := h.queries.GetPartnershipByID(r.Context(), partnershipID)
	if err != nil {
		http.Redirect(w, r, "/recs", http.StatusSeeOther)
		return
	}

	// Determine who is the partner (the other person)
	var partnerUserID int64
	if partnership.UserID == userID {
		if !partnership.PartnerID.Valid {
			http.Redirect(w, r, "/profile", http.StatusSeeOther)
			return
		}
		partnerUserID = partnership.PartnerID.Int64
	} else if partnership.PartnerID.Valid && partnership.PartnerID.Int64 == userID {
		partnerUserID = partnership.UserID
	} else {
		http.Redirect(w, r, "/recs", http.StatusSeeOther)
		return
	}

	partnerUser, err := h.queries.GetUserByID(r.Context(), partnerUserID)
	if err != nil {
		http.Redirect(w, r, "/recs", http.StatusSeeOther)
		return
	}

	myRatings, _ := h.queries.GetUserRatings(r.Context(), userID)
	theirRatings, _ := h.queries.GetUserRatings(r.Context(), partnerUserID)

	myRatingMap := make(map[string]string)
	for _, rating := range myRatings {
		myRatingMap[rating.ShowID] = rating.Rating
	}

	data := TogetherData{
		PageData:        h.basePageDataWithPartners(r, "together"),
		PartnershipID:   partnershipID,
		PartnerName:     partnerUser.Name,
		PartnerInitials: makeInitials(partnerUser.Name),
		MyRatedCount:    len(myRatings),
		TheirRatedCount: len(theirRatings),
	}

	if len(myRatings) < 3 || len(theirRatings) < 3 {
		data.HasEnoughData = false
		if isHTMX(r) {
			renderPartial(w, "together-content", data)
			return
		}
		renderPage(w, "together.html", data)
		return
	}

	data.HasEnoughData = true

	// Together verdict
	verdict, err := h.engine.GetTogetherVerdict(r.Context(), partnershipID, userID, partnerUserID)
	if err == nil {
		data.Verdict = verdict
	}

	// Tonight's recs (together)
	tonight, _ := h.engine.TopNTogether(r.Context(), userID, partnerUserID, 6)
	data.RecsTonight = showsToSWR(tonight, myRatingMap)

	// Because you both liked...
	title, because, _ := h.engine.BecauseYouBothLiked(r.Context(), userID, partnerUserID)
	data.BecauseTitle = title
	data.BecauseShows = showsToSWR(because, myRatingMap)

	// Shows you both liked (overlap)
	overlap, _ := h.queries.GetBothLikedShows(r.Context(), db.GetBothLikedShowsParams{
		UserID:   userID,
		UserID_2: partnerUserID,
	})
	if len(overlap) > 6 {
		overlap = overlap[:6]
	}
	data.OverlapShows = overlap

	if isHTMX(r) {
		renderPartial(w, "together-content", data)
		return
	}
	renderPage(w, "together.html", data)
}

func (h *Handler) TogetherRefresh(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")
	partnershipID, _ := strconv.ParseInt(idStr, 10, 64)

	partnership, err := h.queries.GetPartnershipByID(r.Context(), partnershipID)
	if err != nil {
		return
	}

	var partnerUserID int64
	if partnership.UserID == userID {
		partnerUserID = partnership.PartnerID.Int64
	} else {
		partnerUserID = partnership.UserID
	}

	myRatings, _ := h.queries.GetUserRatings(r.Context(), userID)
	myRatingMap := make(map[string]string)
	for _, rating := range myRatings {
		myRatingMap[rating.ShowID] = rating.Rating
	}

	tonight, _ := h.engine.TopNTogether(r.Context(), userID, partnerUserID, 6)
	renderPartial(w, "poster-row", showsToSWR(tonight, myRatingMap))
}
