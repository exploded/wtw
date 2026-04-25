package tmdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
		ID         int64   `json:"id"`
		PosterPath string  `json:"poster_path"`
		Overview   string  `json:"overview"`
		Popularity float64 `json:"popularity"`
	} `json:"results"`
}

type tvDetail struct {
	ID         int64   `json:"id"`
	PosterPath string  `json:"poster_path"`
	Overview   string  `json:"overview"`
	Popularity float64 `json:"popularity"`
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
	ID         int64
	PosterPath string
	Overview   string
	Popularity float64
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
		ID:         r.ID,
		PosterPath: r.PosterPath,
		Overview:   r.Overview,
		Popularity: r.Popularity,
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
