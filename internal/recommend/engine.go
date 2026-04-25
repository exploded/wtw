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
		case "favourite":
			weight = 2.0
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

	favourites, _ := e.queries.GetFavouriteShows(ctx, userID)
	liked, err := e.queries.GetLikedShows(ctx, userID)
	if err != nil || len(liked) == 0 {
		return e.fallbackVerdict(ctx, userID)
	}
	disliked, _ := e.queries.GetDislikedShows(ctx, userID)

	top, err := e.TopN(ctx, userID, 5)
	if err != nil || len(top) == 0 {
		return e.fallbackVerdict(ctx, userID)
	}

	favouriteTitles := titlesStr(favourites)
	likedTitles := titlesStr(liked)
	dislikedTitles := titlesStr(disliked)
	candidateTitles := titlesStr(top)

	prompt := fmt.Sprintf(
		"You are an opinionated cinephile recommending TV series. The user's absolute favourites: %s. They also liked: %s. They disliked: %s. "+
			"Recommend exactly ONE TV series from this candidate list: %s. "+
			"Respond with valid JSON only, no markdown: "+
			`{"pick":"<exact title>","headline":"You should watch <Title>.","verdict":"<2 sentences, max 50 words, in the voice of a knowledgeable film-friend explaining why>"}`,
		favouriteTitles, likedTitles, dislikedTitles, candidateTitles,
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

// --- Together mode ---

func (e *Engine) ScoreShowsTogether(ctx context.Context, userA, userB int64) ([]ScoredShow, error) {
	ratingsA, err := e.queries.GetUserRatingsWithShows(ctx, userA)
	if err != nil {
		return nil, err
	}
	ratingsB, err := e.queries.GetUserRatingsWithShows(ctx, userB)
	if err != nil {
		return nil, err
	}

	// Build preference vectors for both
	prefA := make(map[string]float64)
	for _, r := range ratingsA {
		gv := genreVector(r.Genre.String)
		weight := 0.0
		if r.Rating == "favourite" {
			weight = 2.0
		} else if r.Rating == "liked" {
			weight = 1.0
		} else if r.Rating == "disliked" {
			weight = -1.0
		}
		for tag, val := range gv {
			prefA[tag] += val * weight
		}
	}
	prefB := make(map[string]float64)
	for _, r := range ratingsB {
		gv := genreVector(r.Genre.String)
		weight := 0.0
		if r.Rating == "favourite" {
			weight = 2.0
		} else if r.Rating == "liked" {
			weight = 1.0
		} else if r.Rating == "disliked" {
			weight = -1.0
		}
		for tag, val := range gv {
			prefB[tag] += val * weight
		}
	}

	// Merge: average both vectors
	merged := make(map[string]float64)
	allTags := make(map[string]bool)
	for t := range prefA {
		allTags[t] = true
	}
	for t := range prefB {
		allTags[t] = true
	}
	for tag := range allTags {
		merged[tag] = (prefA[tag] + prefB[tag]) / 2
	}

	// Build set of disliked shows by either user
	disliked := make(map[string]bool)
	ratedByA := make(map[string]bool)
	ratedByB := make(map[string]bool)
	for _, r := range ratingsA {
		ratedByA[r.ShowID] = true
		if r.Rating == "disliked" {
			disliked[r.ShowID] = true
		}
	}
	for _, r := range ratingsB {
		ratedByB[r.ShowID] = true
		if r.Rating == "disliked" {
			disliked[r.ShowID] = true
		}
	}

	// Score shows not disliked by either, not already watched by both
	allShows, err := e.queries.GetShows(ctx)
	if err != nil {
		return nil, err
	}

	var scored []ScoredShow
	for _, show := range allShows {
		if disliked[show.ID] {
			continue
		}
		// Skip if both have already rated it (nothing new to watch)
		if ratedByA[show.ID] && ratedByB[show.ID] {
			continue
		}

		gv := genreVector(show.Genre.String)
		score := 0.0
		for tag, val := range gv {
			score += val * merged[tag]
		}
		score += show.Popularity.Float64 / 1000
		scored = append(scored, ScoredShow{Show: show, Score: score})
	}

	// Sort descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	return scored, nil
}

func (e *Engine) TopNTogether(ctx context.Context, userA, userB int64, n int) ([]db.Show, error) {
	scored, err := e.ScoreShowsTogether(ctx, userA, userB)
	if err != nil {
		return nil, err
	}
	var result []db.Show
	for i := 0; i < n && i < len(scored); i++ {
		result = append(result, scored[i].Show)
	}
	return result, nil
}

func (e *Engine) BecauseYouBothLiked(ctx context.Context, userA, userB int64) (string, []db.Show, error) {
	bothLiked, err := e.queries.GetBothLikedShows(ctx, db.GetBothLikedShowsParams{
		UserID:   userA,
		UserID_2: userB,
	})
	if err != nil || len(bothLiked) == 0 {
		return "", nil, err
	}

	anchor := bothLiked[0]
	anchorGenre := genreVector(anchor.Genre.String)

	scored, _ := e.ScoreShowsTogether(ctx, userA, userB)
	var matches []db.Show
	for _, ss := range scored {
		gv := genreVector(ss.Show.Genre.String)
		for tag := range anchorGenre {
			if gv[tag] > 0 {
				matches = append(matches, ss.Show)
				break
			}
		}
		if len(matches) >= 6 {
			break
		}
	}

	return anchor.Title, matches, nil
}

func (e *Engine) GetTogetherVerdict(ctx context.Context, partnershipID, userA, userB int64) (*Verdict, error) {
	// Check cache
	cached, err := e.queries.GetPartnerVerdict(ctx, partnershipID)
	if err == nil {
		show, _ := e.queries.GetShowByID(ctx, cached.ShowID.String)
		return &Verdict{
			Headline: cached.Headline,
			Body:     cached.Verdict,
			ShowID:   cached.ShowID.String,
			Show:     &show,
		}, nil
	}

	return e.generateTogetherVerdict(ctx, partnershipID, userA, userB)
}

func (e *Engine) generateTogetherVerdict(ctx context.Context, partnershipID, userA, userB int64) (*Verdict, error) {
	if e.anthropicKey == "" {
		return e.fallbackTogetherVerdict(ctx, partnershipID, userA, userB)
	}

	favouritesA, _ := e.queries.GetFavouriteShows(ctx, userA)
	likedA, _ := e.queries.GetLikedShows(ctx, userA)
	dislikedA, _ := e.queries.GetDislikedShows(ctx, userA)
	favouritesB, _ := e.queries.GetFavouriteShows(ctx, userB)
	likedB, _ := e.queries.GetLikedShows(ctx, userB)
	dislikedB, _ := e.queries.GetDislikedShows(ctx, userB)

	top, err := e.TopNTogether(ctx, userA, userB, 5)
	if err != nil || len(top) == 0 {
		return e.fallbackTogetherVerdict(ctx, partnershipID, userA, userB)
	}

	prompt := fmt.Sprintf(
		"You are an opinionated cinephile recommending a TV series for two people to watch together. "+
			"Person A's favourites: %s. Person A also liked: %s. Person A disliked: %s. "+
			"Person B's favourites: %s. Person B also liked: %s. Person B disliked: %s. "+
			"Recommend exactly ONE series from this candidate list that they would BOTH enjoy: %s. "+
			"Respond with valid JSON only, no markdown: "+
			`{"pick":"<exact title>","headline":"Tonight, watch <Title> together.","verdict":"<2 sentences, max 50 words, explaining why this works for both>"}`,
		titlesStr(favouritesA), titlesStr(likedA), titlesStr(dislikedA),
		titlesStr(favouritesB), titlesStr(likedB), titlesStr(dislikedB),
		titlesStr(top),
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
		slog.Error("claude api error (together)", "error", err)
		return e.fallbackTogetherVerdict(ctx, partnershipID, userA, userB)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Error("claude api non-200 (together)", "status", resp.StatusCode)
		return e.fallbackTogetherVerdict(ctx, partnershipID, userA, userB)
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil || len(apiResp.Content) == 0 {
		return e.fallbackTogetherVerdict(ctx, partnershipID, userA, userB)
	}

	var verdictResp struct {
		Pick     string `json:"pick"`
		Headline string `json:"headline"`
		Verdict  string `json:"verdict"`
	}
	if err := json.Unmarshal([]byte(apiResp.Content[0].Text), &verdictResp); err != nil {
		slog.Error("failed to parse together verdict json", "error", err)
		return e.fallbackTogetherVerdict(ctx, partnershipID, userA, userB)
	}

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

	_ = e.queries.UpsertPartnerVerdict(ctx, db.UpsertPartnerVerdictParams{
		PartnershipID: partnershipID,
		Verdict:       verdictResp.Verdict,
		Headline:      verdictResp.Headline,
		ShowID:        db.NewNullString(showID),
	})

	show, _ := e.queries.GetShowByID(ctx, showID)
	return &Verdict{
		Headline: verdictResp.Headline,
		Body:     verdictResp.Verdict,
		ShowID:   showID,
		Show:     &show,
	}, nil
}

func (e *Engine) fallbackTogetherVerdict(ctx context.Context, partnershipID, userA, userB int64) (*Verdict, error) {
	top, err := e.TopNTogether(ctx, userA, userB, 1)
	if err != nil || len(top) == 0 {
		return nil, fmt.Errorf("no shows to recommend together")
	}
	show := top[0]
	headline := fmt.Sprintf("Tonight, watch %s together.", show.Title)
	body := fmt.Sprintf("Based on both your tastes, %s looks like a great pick for a shared watch.", show.Title)

	_ = e.queries.UpsertPartnerVerdict(ctx, db.UpsertPartnerVerdictParams{
		PartnershipID: partnershipID,
		Verdict:       body,
		Headline:      headline,
		ShowID:        db.NewNullString(show.ID),
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
