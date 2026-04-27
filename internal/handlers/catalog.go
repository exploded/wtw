package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"wtw/internal/db"
	"wtw/internal/middleware"
	"wtw/internal/tmdb"
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
	shows, err := h.queries.GetShows(r.Context())
	if err != nil {
		slog.Error("failed to get shows", "error", err)
	}
	ratings, err := h.queries.GetUserRatings(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get user ratings", "error", err)
	}

	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	counts, err := h.queries.CountRatings(r.Context(), userID)
	if err != nil {
		slog.Error("failed to count ratings", "error", err)
	}

	filter := r.URL.Query().Get("filter")
	search := strings.ToLower(r.URL.Query().Get("q"))

	filtered := filterShows(shows, ratingMap, filter, search)

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
	shows, err := h.queries.GetShows(r.Context())
	if err != nil {
		slog.Error("failed to get shows", "error", err)
	}
	ratings, err := h.queries.GetUserRatings(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get user ratings", "error", err)
	}

	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	filter := r.URL.Query().Get("filter")
	search := strings.ToLower(r.URL.Query().Get("q"))

	filtered := filterShows(shows, ratingMap, filter, search)
	renderPartial(w, "poster-grid", filtered)
}

func filterShows(shows []db.Show, ratingMap map[string]string, filter, search string) []ShowWithRating {
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
	return filtered
}

type TMDBSearchResult struct {
	tmdb.TVResult
	AlreadyInDB bool
}

func (h *Handler) CatalogSearchTMDB(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	results, err := h.tmdbClient.SearchTVMulti(r.Context(), query)
	if err != nil {
		slog.Error("TMDB search failed", "error", err)
		renderPartial(w, "tmdb-results", []TMDBSearchResult{})
		return
	}

	var out []TMDBSearchResult
	for _, res := range results {
		alreadyIn := false
		if res.TmdbID > 0 {
			_, err := h.queries.GetShowByTmdbID(r.Context(), sql.NullInt64{Int64: res.TmdbID, Valid: true})
			if err == nil {
				alreadyIn = true
			}
		}
		out = append(out, TMDBSearchResult{TVResult: res, AlreadyInDB: alreadyIn})
	}

	renderPartial(w, "tmdb-results", out)
}

func (h *Handler) CatalogAddTMDB(w http.ResponseWriter, r *http.Request) {
	tmdbIDStr := r.PathValue("tmdb_id")
	tmdbID, err := strconv.ParseInt(tmdbIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid tmdb_id", http.StatusBadRequest)
		return
	}

	show, err := h.tmdbClient.EnsureShow(r.Context(), tmdbID)
	if err != nil {
		http.Error(w, "failed to add show", http.StatusInternalServerError)
		return
	}

	userID := middleware.GetUserID(r.Context())
	ratings, _ := h.queries.GetUserRatings(r.Context(), userID)
	ratingMap := make(map[string]string)
	for _, rating := range ratings {
		ratingMap[rating.ShowID] = rating.Rating
	}

	renderPartial(w, "poster-card", ShowWithRating{Show: *show, Rating: ratingMap[show.ID]})
}
