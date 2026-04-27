package handlers

import (
	"database/sql"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"wtw/internal/db"
	"wtw/internal/middleware"
)

type ActivePartner struct {
	PartnershipID int64
	Name          string
	Email         string
	Initials      string
}

type PendingPartner struct {
	PartnershipID int64
	Email         string
}

type IncomingRequest struct {
	PartnershipID int64
	Name          string
	Email         string
}

type ProfileData struct {
	PageData
	Favourite        int64
	Liked            int64
	Disliked         int64
	Total            int64
	LikedShows       []db.Show
	TopShows         []db.Show
	ShowCount        int
	MemberSince      string
	TasteSummary     string
	ActivePartners   []ActivePartner
	PendingPartners  []PendingPartner
	IncomingRequests []IncomingRequest
	PartnerError     string
	PartnerSuccess   string
}

func buildTasteSummary(ratings []db.GetUserRatingsWithShowsRow) string {
	if len(ratings) == 0 {
		return ""
	}

	likedGenres := map[string]int{}
	dislikedGenres := map[string]int{}
	var totalLiked, totalDisliked int

	for _, r := range ratings {
		genre := r.Genre.String
		if genre == "" {
			continue
		}
		switch r.Rating {
		case "favourite", "liked":
			likedGenres[genre]++
			totalLiked++
		case "disliked":
			dislikedGenres[genre]++
			totalDisliked++
		}
	}

	if totalLiked == 0 {
		return ""
	}

	type genreCount struct {
		name  string
		count int
	}
	ranked := make([]genreCount, 0, len(likedGenres))
	for g, c := range likedGenres {
		ranked = append(ranked, genreCount{g, c})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].count != ranked[j].count {
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].name < ranked[j].name
	})

	// Top genres the user enjoys
	topN := ranked
	if len(topN) > 3 {
		topN = topN[:3]
	}
	names := make([]string, len(topN))
	for i, g := range topN {
		names[i] = strings.ToLower(g.name)
	}

	var parts []string
	switch len(names) {
	case 1:
		parts = append(parts, fmt.Sprintf("You're drawn to %s", names[0]))
	case 2:
		parts = append(parts, fmt.Sprintf("You're drawn to %s and %s", names[0], names[1]))
	default:
		parts = append(parts, fmt.Sprintf("You're drawn to %s, %s, and %s", names[0], names[1], names[2]))
	}

	if len(ranked) > 4 {
		parts[0] += ", with broad taste across many genres"
	}

	// Genres the user tends to avoid
	if totalDisliked > 0 {
		avoided := make([]genreCount, 0, len(dislikedGenres))
		for g, c := range dislikedGenres {
			if likedGenres[g] == 0 || c > likedGenres[g] {
				avoided = append(avoided, genreCount{g, c})
			}
		}
		sort.Slice(avoided, func(i, j int) bool {
			return avoided[i].count > avoided[j].count
		})
		if len(avoided) > 2 {
			avoided = avoided[:2]
		}
		if len(avoided) > 0 {
			avoidNames := make([]string, len(avoided))
			for i, g := range avoided {
				avoidNames[i] = strings.ToLower(g.name)
			}
			if len(avoidNames) == 1 {
				parts = append(parts, fmt.Sprintf("and tend to skip %s", avoidNames[0]))
			} else {
				parts = append(parts, fmt.Sprintf("and tend to skip %s and %s", avoidNames[0], avoidNames[1]))
			}
		}
	}

	return strings.Join(parts, ", ") + "."
}

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userEmail := middleware.GetUserEmail(r.Context())

	counts, err := h.queries.CountRatings(r.Context(), userID)
	if err != nil {
		slog.Error("failed to count ratings", "error", err)
	}
	favourite := int64(counts.Favourite.Float64)
	liked := int64(counts.Liked.Float64)
	disliked := int64(counts.Disliked.Float64)

	likedShows, err := h.queries.GetLikedShows(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get liked shows", "error", err)
	}
	shows, err := h.queries.GetShows(r.Context())
	if err != nil {
		slog.Error("failed to get shows", "error", err)
	}

	topShows := likedShows
	if len(topShows) > 5 {
		topShows = topShows[:5]
	}

	user, err := h.queries.GetUserByID(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get user", "error", err)
	}
	ratingsWithShows, err := h.queries.GetUserRatingsWithShows(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get ratings with shows", "error", err)
	}

	data := ProfileData{
		PageData:     h.basePageDataWithPartners(r, "profile"),
		Favourite:    favourite,
		Liked:        liked,
		Disliked:     disliked,
		Total:        counts.Total,
		LikedShows:   likedShows,
		TopShows:     topShows,
		ShowCount:    len(shows),
		MemberSince:  user.CreatedAt.Format("January 2006"),
		TasteSummary: buildTasteSummary(ratingsWithShows),
	}

	// Active partnerships (as owner)
	seen := map[string]bool{}
	owned, err := h.queries.GetActivePartnershipsAsOwner(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get owned partnerships", "error", err)
	}
	for _, p := range owned {
		seen[p.PartnerEmail] = true
		data.ActivePartners = append(data.ActivePartners, ActivePartner{
			PartnershipID: p.ID,
			Name:          p.PartnerName.String,
			Email:         p.PartnerEmail,
			Initials:      makeInitials(p.PartnerName.String),
		})
	}

	// Active partnerships (as partner) - skip if already seen from owned
	asPartner, err := h.queries.GetActivePartnershipsAsPartner(r.Context(), db.NewNullInt64(userID))
	if err != nil {
		slog.Error("failed to get partner partnerships", "error", err)
	}
	for _, p := range asPartner {
		if seen[p.PartnerEmail] {
			continue
		}
		data.ActivePartners = append(data.ActivePartners, ActivePartner{
			PartnershipID: p.ID,
			Name:          p.PartnerName.String,
			Email:         p.PartnerEmail,
			Initials:      makeInitials(p.PartnerName.String),
		})
	}

	// Pending (sent by me)
	pending, err := h.queries.GetPendingPartnershipsByUser(r.Context(), userID)
	if err != nil {
		slog.Error("failed to get pending partnerships", "error", err)
	}
	for _, p := range pending {
		data.PendingPartners = append(data.PendingPartners, PendingPartner{
			PartnershipID: p.ID,
			Email:         p.PartnerEmail,
		})
	}

	// Incoming requests (others who added my email)
	incoming, err := h.queries.GetIncomingPartnerRequests(r.Context(), userEmail)
	if err != nil {
		slog.Error("failed to get incoming requests", "error", err)
	}
	for _, p := range incoming {
		data.IncomingRequests = append(data.IncomingRequests, IncomingRequest{
			PartnershipID: p.ID,
			Name:          p.RequesterName,
			Email:         p.RequesterEmail,
		})
	}

	if isHTMX(r) {
		renderPartial(w, "profile-content", data)
		return
	}
	renderPage(w, "profile.html", data)
}

func (h *Handler) AddPartner(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userEmail := middleware.GetUserEmail(r.Context())
	partnerEmail := strings.TrimSpace(r.FormValue("email"))

	if partnerEmail == "" {
		h.profileWithError(w, r, "Please enter an email address.")
		return
	}
	if strings.EqualFold(partnerEmail, userEmail) {
		h.profileWithError(w, r, "You can't add yourself as a partner.")
		return
	}

	// Check if partnership already exists
	pending, _ := h.queries.GetPendingPartnershipsByUser(r.Context(), userID)
	for _, p := range pending {
		if strings.EqualFold(p.PartnerEmail, partnerEmail) {
			h.profileWithError(w, r, "You already have a pending invite to this email.")
			return
		}
	}
	owned, _ := h.queries.GetActivePartnershipsAsOwner(r.Context(), userID)
	for _, p := range owned {
		if strings.EqualFold(p.PartnerEmail, partnerEmail) {
			h.profileWithError(w, r, "You're already linked with this partner.")
			return
		}
	}

	// Check if partner user exists
	var partnerID sql.NullInt64
	partnerUser, err := h.queries.GetUserByEmail(r.Context(), partnerEmail)
	if err == nil {
		partnerID = sql.NullInt64{Int64: partnerUser.ID, Valid: true}
	}

	// Create partnership
	_, err = h.queries.CreatePartnership(r.Context(), db.CreatePartnershipParams{
		UserID:       userID,
		PartnerEmail: partnerEmail,
		PartnerID:    partnerID,
		Status:       "pending",
	})
	if err != nil {
		h.profileWithError(w, r, "Failed to create partnership.")
		return
	}

	// Check if the other person already added us - auto-activate both
	if partnerID.Valid {
		incoming, _ := h.queries.GetIncomingPartnerRequests(r.Context(), userEmail)
		for _, req := range incoming {
			if req.UserID == partnerUser.ID {
				// They already added us! Activate both
				_ = h.queries.ActivatePartnership(r.Context(), db.ActivatePartnershipParams{
					PartnerID: sql.NullInt64{Int64: userID, Valid: true},
					ID:        req.ID,
				})
				// Also activate ours
				myPending, _ := h.queries.GetPendingPartnershipsByUser(r.Context(), userID)
				for _, mp := range myPending {
					if strings.EqualFold(mp.PartnerEmail, partnerEmail) {
						_ = h.queries.ActivatePartnership(r.Context(), db.ActivatePartnershipParams{
							PartnerID: partnerID,
							ID:        mp.ID,
						})
					}
				}
				h.profileWithSuccess(w, r, fmt.Sprintf("Linked with %s!", html.EscapeString(partnerUser.Name)))
				return
			}
		}
	}

	h.profileWithSuccess(w, r, fmt.Sprintf("Invite sent to %s. They'll need to add your email too.", html.EscapeString(partnerEmail)))
}

func (h *Handler) AcceptPartner(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	userEmail := middleware.GetUserEmail(r.Context())
	idStr := r.PathValue("id")
	partnershipID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	partnership, err := h.queries.GetPartnershipByID(r.Context(), partnershipID)
	if err != nil {
		h.profileWithError(w, r, "Partnership not found.")
		return
	}

	// Authorization: only the intended partner can accept
	if !strings.EqualFold(partnership.PartnerEmail, userEmail) {
		h.profileWithError(w, r, "You are not authorized to accept this partnership.")
		return
	}

	// Activate their request (set us as the partner)
	_ = h.queries.ActivatePartnership(r.Context(), db.ActivatePartnershipParams{
		PartnerID: sql.NullInt64{Int64: userID, Valid: true},
		ID:        partnershipID,
	})

	// Look up the requester's email for the reverse partnership
	requester, err := h.queries.GetUserByID(r.Context(), partnership.UserID)
	if err != nil {
		slog.Error("failed to look up requester for reverse partnership", "error", err)
		h.profileWithSuccess(w, r, "Partnership accepted!")
		return
	}

	// Create our side of the partnership too (active immediately)
	_, _ = h.queries.CreatePartnership(r.Context(), db.CreatePartnershipParams{
		UserID:       userID,
		PartnerEmail: requester.Email,
		PartnerID:    sql.NullInt64{Int64: partnership.UserID, Valid: true},
		Status:       "active",
	})

	h.profileWithSuccess(w, r, "Partnership accepted!")
}

func (h *Handler) RemovePartner(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	idStr := r.PathValue("id")
	partnershipID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Authorization: only a party to the partnership can remove it
	partnership, err := h.queries.GetPartnershipByID(r.Context(), partnershipID)
	if err != nil {
		http.Error(w, "partnership not found", http.StatusNotFound)
		return
	}
	if partnership.UserID != userID && (!partnership.PartnerID.Valid || partnership.PartnerID.Int64 != userID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	_ = h.queries.DeletePartnership(r.Context(), partnershipID)

	// Redirect back to profile
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/profile")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *Handler) profileWithError(w http.ResponseWriter, r *http.Request, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<p class="partner-msg partner-msg--error">%s</p>`, html.EscapeString(msg))
}

func (h *Handler) profileWithSuccess(w http.ResponseWriter, r *http.Request, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<p class="partner-msg partner-msg--ok">%s</p>`, html.EscapeString(msg))
}
