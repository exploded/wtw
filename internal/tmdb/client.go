package tmdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"wtw/internal/db"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
	queries    *db.Queries
}

func NewClient(apiKey string, queries *db.Queries) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		queries:    queries,
	}
}

type searchResult struct {
	Results []struct {
		ID           int64   `json:"id"`
		Name         string  `json:"name"`
		PosterPath   string  `json:"poster_path"`
		Overview     string  `json:"overview"`
		Popularity   float64 `json:"popularity"`
		FirstAirDate string  `json:"first_air_date"`
		GenreIDs     []int   `json:"genre_ids"`
	} `json:"results"`
}

type tmdbGenre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tvDetail struct {
	ID           int64       `json:"id"`
	Name         string      `json:"name"`
	PosterPath   string      `json:"poster_path"`
	Overview     string      `json:"overview"`
	Popularity   float64     `json:"popularity"`
	FirstAirDate string      `json:"first_air_date"`
	Genres       []tmdbGenre `json:"genres"`
}

// RefreshAllPosters searches TMDB for each show and updates poster_path,
// synopsis, tmdb_id, and popularity. Called once at startup.
func (c *Client) RefreshAllPosters(ctx context.Context) {
	if c.apiKey == "" {
		slog.Info("TMDB_API_KEY not set, skipping poster refresh")
		return
	}

	shows, err := c.queries.GetShows(ctx)
	if err != nil {
		slog.Error("failed to get shows for refresh", "error", err)
		return
	}

	updated := 0
	for _, show := range shows {
		// If we already have a tmdb_id, fetch directly
		if show.TmdbID.Valid {
			detail, err := c.fetchByID(ctx, show.TmdbID.Int64)
			if err == nil && detail.PosterPath != "" {
				c.updateShow(ctx, show.ID, detail)
				updated++
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// Otherwise search by title
		result, err := c.searchTV(ctx, show.Title)
		if err != nil {
			slog.Warn("tmdb search failed", "title", show.Title, "error", err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if result != nil {
			c.updateShow(ctx, show.ID, &tvDetail{
				ID:         result.ID,
				PosterPath: result.PosterPath,
				Overview:   result.Overview,
				Popularity: result.Popularity,
			})
			updated++
		}

		time.Sleep(200 * time.Millisecond)
	}

	slog.Info("tmdb poster refresh complete", "updated", updated, "total", len(shows))
}

type searchHit struct {
	ID           int64
	Name         string
	PosterPath   string
	Overview     string
	Popularity   float64
	FirstAirDate string
	GenreIDs     []int
}

func (c *Client) searchTV(ctx context.Context, title string) (*searchHit, error) {
	u := fmt.Sprintf("https://api.themoviedb.org/3/search/tv?api_key=%s&query=%s&page=1",
		c.apiKey, url.QueryEscape(title))

	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var sr searchResult
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	if len(sr.Results) == 0 {
		return nil, nil
	}

	r := sr.Results[0]
	return &searchHit{
		ID:           r.ID,
		Name:         r.Name,
		PosterPath:   r.PosterPath,
		Overview:     r.Overview,
		Popularity:   r.Popularity,
		FirstAirDate: r.FirstAirDate,
		GenreIDs:     r.GenreIDs,
	}, nil
}

func (c *Client) fetchByID(ctx context.Context, tmdbID int64) (*tvDetail, error) {
	u := fmt.Sprintf("https://api.themoviedb.org/3/tv/%d?api_key=%s", tmdbID, c.apiKey)

	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var detail tvDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func (c *Client) updateShow(ctx context.Context, showID string, detail *tvDetail) {
	err := c.queries.UpdateShowMeta(ctx, db.UpdateShowMetaParams{
		PosterPath: db.NewNullString(detail.PosterPath),
		Synopsis:   db.NewNullString(detail.Overview),
		TmdbID:     sql.NullInt64{Int64: detail.ID, Valid: detail.ID > 0},
		Popularity: sql.NullFloat64{Float64: detail.Popularity, Valid: true},
		ID:         showID,
	})
	if err != nil {
		slog.Error("failed to update show meta", "show", showID, "error", err)
	}
}

func (c *Client) StartRefreshLoop(ctx context.Context) {
	go func() {
		c.SeedFromTMDB(ctx)
		c.RefreshAllPosters(ctx)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.RefreshAllPosters(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// TVResult is an exported type for TMDB search results used by handlers.
type TVResult struct {
	TmdbID       int64
	Name         string
	PosterPath   string
	Overview     string
	Popularity   float64
	Year         string
	Genre        string
}

// Genre name mapping from TMDB to our simpler tags.
var tmdbGenreMap = map[string]string{
	"Sci-Fi & Fantasy":    "Sci-fi",
	"Action & Adventure":  "Action",
	"War & Politics":      "War drama",
	"Crime":               "Crime",
	"Comedy":              "Comedy",
	"Drama":               "Drama",
	"Mystery":             "Mystery",
	"Documentary":         "Documentary",
	"Animation":           "Animated",
	"Family":              "Family",
	"Western":             "Western",
	"Reality":             "Reality",
	"Soap":                "Drama",
	"Talk":                "Talk",
	"News":                "News",
	"Kids":                "Family",
}

// TMDB genre ID to name mapping (TV genres).
var tmdbGenreIDMap = map[int]string{
	10759: "Action & Adventure",
	16:    "Animation",
	35:    "Comedy",
	80:    "Crime",
	99:    "Documentary",
	18:    "Drama",
	10751: "Family",
	10762: "Kids",
	9648:  "Mystery",
	10763: "News",
	10764: "Reality",
	10765: "Sci-Fi & Fantasy",
	10766: "Soap",
	10767: "Talk",
	10768: "War & Politics",
	37:    "Western",
}

func genreString(genres []tmdbGenre) string {
	var parts []string
	seen := make(map[string]bool)
	for _, g := range genres {
		mapped := g.Name
		if m, ok := tmdbGenreMap[g.Name]; ok {
			mapped = m
		}
		lower := strings.ToLower(mapped)
		if !seen[lower] {
			seen[lower] = true
			parts = append(parts, mapped)
		}
	}
	return strings.Join(parts, " ")
}

func genreStringFromIDs(ids []int) string {
	var genres []tmdbGenre
	for _, id := range ids {
		if name, ok := tmdbGenreIDMap[id]; ok {
			genres = append(genres, tmdbGenre{ID: id, Name: name})
		}
	}
	return genreString(genres)
}

func yearFromDate(firstAirDate string) string {
	if len(firstAirDate) >= 4 {
		return firstAirDate[:4]
	}
	return ""
}

// SearchTVMulti returns up to 20 TMDB search results for a query.
func (c *Client) SearchTVMulti(ctx context.Context, query string) ([]TVResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not set")
	}

	u := fmt.Sprintf("https://api.themoviedb.org/3/search/tv?api_key=%s&query=%s&page=1",
		c.apiKey, url.QueryEscape(query))

	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var sr searchResult
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	var results []TVResult
	for _, r := range sr.Results {
		results = append(results, TVResult{
			TmdbID:     r.ID,
			Name:       r.Name,
			PosterPath: r.PosterPath,
			Overview:   r.Overview,
			Popularity: r.Popularity,
			Year:       yearFromDate(r.FirstAirDate),
			Genre:      genreStringFromIDs(r.GenreIDs),
		})
	}
	return results, nil
}

// FetchPopularTV returns one page of popular TV shows from TMDB.
func (c *Client) FetchPopularTV(ctx context.Context, page int) ([]TVResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not set")
	}

	u := fmt.Sprintf("https://api.themoviedb.org/3/tv/popular?api_key=%s&page=%d", c.apiKey, page)

	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var sr searchResult
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	var results []TVResult
	for _, r := range sr.Results {
		results = append(results, TVResult{
			TmdbID:     r.ID,
			Name:       r.Name,
			PosterPath: r.PosterPath,
			Overview:   r.Overview,
			Popularity: r.Popularity,
			Year:       yearFromDate(r.FirstAirDate),
			Genre:      genreStringFromIDs(r.GenreIDs),
		})
	}
	return results, nil
}

// EnsureShow checks if a show with the given TMDB ID exists in the DB.
// If not, it fetches details from TMDB and inserts it.
func (c *Client) EnsureShow(ctx context.Context, tmdbID int64) (*db.Show, error) {
	existing, err := c.queries.GetShowByTmdbID(ctx, sql.NullInt64{Int64: tmdbID, Valid: true})
	if err == nil {
		return &existing, nil
	}

	detail, err := c.fetchByID(ctx, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("fetching TMDB show %d: %w", tmdbID, err)
	}

	slug := db.Slugify(detail.Name)
	year := yearFromDate(detail.FirstAirDate)
	genre := genreString(detail.Genres)

	// Handle slug collision by appending year
	if _, err := c.queries.GetShowByID(ctx, slug); err == nil {
		slug = slug + "-" + year
	}

	err = c.queries.InsertShow(ctx, db.InsertShowParams{
		ID:         slug,
		TmdbID:     sql.NullInt64{Int64: detail.ID, Valid: true},
		Title:      detail.Name,
		Year:       year,
		PosterPath: db.NewNullString(detail.PosterPath),
		Genre:      db.NewNullString(genre),
		Synopsis:   db.NewNullString(detail.Overview),
		Popularity: sql.NullFloat64{Float64: detail.Popularity, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("inserting show %s: %w", slug, err)
	}

	show, err := c.queries.GetShowByID(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &show, nil
}

// EnsureShowByTitle searches TMDB for a title and ensures the first result is in the DB.
func (c *Client) EnsureShowByTitle(ctx context.Context, title string) (*db.Show, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("TMDB API key not set")
	}

	hit, err := c.searchTV(ctx, title)
	if err != nil {
		return nil, err
	}
	if hit == nil {
		return nil, fmt.Errorf("no TMDB results for %q", title)
	}

	return c.EnsureShow(ctx, hit.ID)
}

// SeedFromTMDB imports popular shows from TMDB if the catalog is small.
func (c *Client) SeedFromTMDB(ctx context.Context) {
	if c.apiKey == "" {
		slog.Info("TMDB_API_KEY not set, skipping TMDB seed")
		return
	}

	count, err := c.queries.CountShows(ctx)
	if err != nil || count >= 100 {
		if err == nil {
			slog.Info("catalog already has enough shows, skipping TMDB seed", "count", count)
		}
		return
	}

	slog.Info("seeding catalog from TMDB popular shows", "current_count", count)

	inserted := 0
	for page := 1; page <= 10; page++ {
		results, err := c.FetchPopularTV(ctx, page)
		if err != nil {
			slog.Warn("tmdb popular page failed", "page", page, "error", err)
			continue
		}

		for _, r := range results {
			// Fetch full details for genres
			detail, err := c.fetchByID(ctx, r.TmdbID)
			if err != nil {
				slog.Warn("tmdb fetch detail failed", "tmdb_id", r.TmdbID, "error", err)
				time.Sleep(200 * time.Millisecond)
				continue
			}

			slug := db.Slugify(detail.Name)
			year := yearFromDate(detail.FirstAirDate)
			genre := genreString(detail.Genres)

			if _, err := c.queries.GetShowByID(ctx, slug); err == nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			err = c.queries.InsertShow(ctx, db.InsertShowParams{
				ID:         slug,
				TmdbID:     sql.NullInt64{Int64: detail.ID, Valid: true},
				Title:      detail.Name,
				Year:       year,
				PosterPath: db.NewNullString(detail.PosterPath),
				Genre:      db.NewNullString(genre),
				Synopsis:   db.NewNullString(detail.Overview),
				Popularity: sql.NullFloat64{Float64: detail.Popularity, Valid: true},
			})
			if err == nil {
				inserted++
			}

			time.Sleep(200 * time.Millisecond)
		}
	}

	slog.Info("tmdb seed complete", "inserted", inserted)
}
