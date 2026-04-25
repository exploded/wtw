package recommend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"

	"wtw/internal/db"
)

type Engine struct {
	queries      *db.Queries
	anthropicKey string
}

func NewEngine(queries *db.Queries, anthropicKey string) *Engine {
	return &Engine{queries: queries, anthropicKey: anthropicKey}
}

// Genre tags used for scoring
var genreTags = []string{
	"Crime", "Drama", "Sci-fi", "Comedy", "Fantasy", "Period",
	"Mystery", "Thriller", "Animated", "War", "Western", "Spy", "Limited",
}

func genreVector(genre string) map[string]float64 {
	v := make(map[string]float64)
	lower := strings.ToLower(genre)
	for _, tag := range genreTags {
		if strings.Contains(lower, strings.ToLower(tag)) {
			v[tag] = 1
		}
	}
	// Also handle compound genres
	if strings.Contains(lower, "dark comedy") {
		v["Comedy"] = 1
		v["Drama"] = 1
	}
	if strings.Contains(lower, "comedy drama") {
		v["Comedy"] = 1
		v["Drama"] = 1
	}
	return v
}

type ScoredShow struct {
	Show  db.Show
	Score float64
}

func (e *Engine) ScoreShows(ctx context.Context, userID int64) ([]ScoredShow, error) {
	ratings, err := e.queries.GetUserRatingsWithShows(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build preference vector
	pref := make(map[string]float64)
	for _, r := range ratings {
		gv := genreVector(r.Genre.String)
		weight := 0.0
		switch r.Rating {
		case "liked":
			weight = 1.0
		case "disliked":
			weight = -1.0
		}
		for tag, val := range gv {
			pref[tag] += val * weight
		}
	}

	// Score unrated shows
	unrated, err := e.queries.GetUnratedShows(ctx, userID)
	if err != nil {
		return nil, err
	}

	var scored []ScoredShow
	for _, show := range unrated {
		gv := genreVector(show.Genre.String)
		score := 0.0
		for tag, val := range gv {
			score += val * pref[tag]
		}
		// Small popularity bonus
		score += show.Popularity.Float64 / 1000
		scored = append(scored, ScoredShow{Show: show, Score: score})
	}

	// Sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	return scored, nil
}

func (e *Engine) TopN(ctx context.Context, userID int64, n int) ([]db.Show, error) {
	scored, err := e.ScoreShows(ctx, userID)
	if err != nil {
		return nil, err
	}
	var result []db.Show
	for i := 0; i < n && i < len(scored); i++ {
		result = append(result, scored[i].Show)
	}
	return result, nil
}

func (e *Engine) BecauseYouLiked(ctx context.Context, userID int64) (string, []db.Show, error) {
	liked, err := e.queries.GetLikedShows(ctx, userID)
	if err != nil || len(liked) == 0 {
		return "", nil, err
	}

	// Pick the most recently liked show
	anchor := liked[0]
	anchorGenre := genreVector(anchor.Genre.String)

	// Find unrated shows with matching genre
	unrated, err := e.queries.GetUnratedShows(ctx, userID)
	if err != nil {
		return anchor.Title, nil, err
	}

	var matches []db.Show
	for _, show := range unrated {
		gv := genreVector(show.Genre.String)
		for tag := range anchorGenre {
			if gv[tag] > 0 {
				matches = append(matches, show)
				break
			}
		}
	}

	// Take up to 6
	if len(matches) > 6 {
		matches = matches[:6]
	}

	return anchor.Title, matches, nil
}

func (e *Engine) ComfortRewatches(ctx context.Context, userID int64) ([]db.Show, error) {
	unrated, err := e.queries.GetUnratedShows(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Pick comedy/light shows
	var comfort []db.Show
	for _, show := range unrated {
		lower := strings.ToLower(show.Genre.String)
		if strings.Contains(lower, "comedy") {
			comfort = append(comfort, show)
		}
	}

	// Shuffle and take up to 6
	rand.Shuffle(len(comfort), func(i, j int) {
		comfort[i], comfort[j] = comfort[j], comfort[i]
	})
	if len(comfort) > 6 {
		comfort = comfort[:6]
	}

	return comfort, nil
}

type Verdict struct {
	Headline string
	Body     string
	ShowID   string
	Show     *db.Show
}

func (e *Engine) GetVerdict(ctx context.Context, userID int64) (*Verdict, error) {
	// Check cache first
	cached, err := e.queries.GetVerdict(ctx, userID)
	if err == nil {
		show, _ := e.queries.GetShowByID(ctx, cached.ShowID.String)
		return &Verdict{
			Headline: cached.Headline,
			Body:     cached.Verdict,
			ShowID:   cached.ShowID.String,
			Show:     &show,
		}, nil
	}

	// Generate new verdict
	return e.generateVerdict(ctx, userID)
}

func (e *Engine) generateVerdict(ctx context.Context, userID int64) (*Verdict, error) {
	if e.anthropicKey == "" {
		return e.fallbackVerdict(ctx, userID)
	}

	liked, err := e.queries.GetLikedShows(ctx, userID)
	if err != nil || len(liked) == 0 {
		return e.fallbackVerdict(ctx, userID)
	}
	disliked, _ := e.queries.GetDislikedShows(ctx, userID)

	top, err := e.TopN(ctx, userID, 5)
	if err != nil || len(top) == 0 {
		return e.fallbackVerdict(ctx, userID)
	}

	likedTitles := titlesStr(liked)
	dislikedTitles := titlesStr(disliked)
	candidateTitles := titlesStr(top)

	prompt := fmt.Sprintf(
		"You are an opinionated cinephile recommending TV series. The user liked: %s. They disliked: %s. "+
			"Recommend exactly ONE TV series from this candidate list: %s. "+
			"Respond with valid JSON only, no markdown: "+
			`{"pick":"<exact title>","headline":"You should watch <Title>.","verdict":"<2 sentences, max 50 words, in the voice of a knowledgeable film-friend explaining why>"}`,
		likedTitles, dislikedTitles, candidateTitles,
	)

	body := map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 200,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.anthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("claude api error", "error", err)
		return e.fallbackVerdict(ctx, userID)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Error("claude api non-200", "status", resp.StatusCode)
		return e.fallbackVerdict(ctx, userID)
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || len(apiResp.Content) == 0 {
		return e.fallbackVerdict(ctx, userID)
	}

	var verdictResp struct {
		Pick     string `json:"pick"`
		Headline string `json:"headline"`
		Verdict  string `json:"verdict"`
	}
	if err := json.Unmarshal([]byte(apiResp.Content[0].Text), &verdictResp); err != nil {
		slog.Error("failed to parse verdict json", "error", err, "text", apiResp.Content[0].Text)
		return e.fallbackVerdict(ctx, userID)
	}

	// Find the show ID
	var showID string
	for _, s := range top {
		if strings.EqualFold(s.Title, verdictResp.Pick) {
			showID = s.ID
			break
		}
	}
	if showID == "" && len(top) > 0 {
		showID = top[0].ID
	}

	// Cache it
	_ = e.queries.UpsertVerdict(ctx, db.UpsertVerdictParams{
		UserID:   userID,
		Verdict:  verdictResp.Verdict,
		Headline: verdictResp.Headline,
		ShowID:   db.NewNullString(showID),
	})

	show, _ := e.queries.GetShowByID(ctx, showID)
	return &Verdict{
		Headline: verdictResp.Headline,
		Body:     verdictResp.Verdict,
		ShowID:   showID,
		Show:     &show,
	}, nil
}

func (e *Engine) fallbackVerdict(ctx context.Context, userID int64) (*Verdict, error) {
	top, err := e.TopN(ctx, userID, 1)
	if err != nil || len(top) == 0 {
		return nil, fmt.Errorf("no shows to recommend")
	}
	show := top[0]
	headline := fmt.Sprintf("You should watch %s.", show.Title)
	body := fmt.Sprintf("Based on what you've rated so far, %s looks like a strong match for your taste. Give it a try.", show.Title)

	_ = e.queries.UpsertVerdict(ctx, db.UpsertVerdictParams{
		UserID:   userID,
		Verdict:  body,
		Headline: headline,
		ShowID:   db.NewNullString(show.ID),
	})

	return &Verdict{
		Headline: headline,
		Body:     body,
		ShowID:   show.ID,
		Show:     &show,
	}, nil
}

func titlesStr(shows []db.Show) string {
	var titles []string
	for _, s := range shows {
		titles = append(titles, s.Title)
	}
	if len(titles) == 0 {
		return "none"
	}
	return strings.Join(titles, ", ")
}
