package handlers

import (
	"net/http"
	"strings"

	"wtw/internal/middleware"
)

type CatalogData struct {
	PageData
	Shows     []ShowWithRating
	Favourite int64
	Liked     int64
	Disliked  int64
	Unrated   int64
	Filter    string
	Search    string
	ShowCount int
}

func (h *Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shows, _ := h.queries.GetShows(r.Context())
	ratings, _ := h.queries.GetUserRatings(r.Context(), userID)

	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	counts, _ := h.queries.CountRatings(r.Context(), userID)

	filter := r.URL.Query().Get("filter")
	search := strings.ToLower(r.URL.Query().Get("q"))

	var filtered []ShowWithRating
	for _, show := range shows {
		rating := ratingMap[show.ID]
		swr := ShowWithRating{Show: show, Rating: rating}

		// Apply filter
		if filter != "" && filter != "all" {
			if filter == "unrated" && rating != "" {
				continue
			}
			if filter != "unrated" && rating != filter {
				continue
			}
		}

		// Apply search
		if search != "" && !strings.Contains(strings.ToLower(show.Title), search) {
			continue
		}

		filtered = append(filtered, swr)
	}

	favourite := int64(counts.Favourite.Float64)
	liked := int64(counts.Liked.Float64)
	disliked := int64(counts.Disliked.Float64)

	data := CatalogData{
		PageData:  h.basePageDataWithPartners(r, "catalog"),
		Shows:     filtered,
		Favourite: favourite,
		Liked:     liked,
		Disliked:  disliked,
		Unrated:   int64(len(shows)) - favourite - liked - disliked,
		Filter:    filter,
		Search:    r.URL.Query().Get("q"),
		ShowCount: len(shows),
	}

	if isHTMX(r) {
		renderPartial(w, "catalog-content", data)
		return
	}
	renderPage(w, "catalog.html", data)
}

func (h *Handler) CatalogFilter(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shows, _ := h.queries.GetShows(r.Context())
	ratings, _ := h.queries.GetUserRatings(r.Context(), userID)

	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	filter := r.URL.Query().Get("filter")
	search := strings.ToLower(r.URL.Query().Get("q"))

	var filtered []ShowWithRating
	for _, show := range shows {
		rating := ratingMap[show.ID]

		if filter != "" && filter != "all" {
			if filter == "unrated" && rating != "" {
				continue
			}
			if filter != "unrated" && rating != filter {
				continue
			}
		}

		if search != "" && !strings.Contains(strings.ToLower(show.Title), search) {
			continue
		}

		filtered = append(filtered, ShowWithRating{Show: show, Rating: rating})
	}

	renderPartial(w, "poster-grid", filtered)
}
