package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"

	"wtw/internal/db"
	"wtw/internal/middleware"
	"wtw/internal/recommend"
	"wtw/internal/session"
	"wtw/internal/tmdb"
)

var (
	pageTemplates map[string]*template.Template
	partialTmpl   *template.Template
	tmplOnce      sync.Once
)

func loadTemplates() {
	funcMap := template.FuncMap{
		"initials": func(name string) string {
			if len(name) == 0 {
				return "?"
			}
			result := ""
			words := splitWords(name)
			for _, w := range words {
				if len(w) > 0 {
					result += string([]rune(w)[:1])
				}
				if len(result) >= 2 {
					break
				}
			}
			return result
		},
	}

	// Parse partials first (shared across all pages)
	partialTmpl = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/partials/*.html"))

	// Each page template = layout + partials + page-specific content
	pageTemplates = make(map[string]*template.Template)
	pages := []string{"landing.html", "onboarding.html", "recs.html", "catalog.html", "profile.html", "together.html"}
	for _, page := range pages {
		// Clone partials so each page gets its own template set
		t := template.Must(partialTmpl.Clone())
		t = template.Must(t.ParseFiles(
			"templates/layout.html",
			filepath.Join("templates", page),
		))
		pageTemplates[page] = t
	}
}

func getTemplates() {
	tmplOnce.Do(loadTemplates)
}

func init() {
	getTemplates()
}

func splitWords(s string) []string {
	var words []string
	word := ""
	for _, r := range s {
		if r == ' ' || r == '-' || r == '.' {
			if word != "" {
				words = append(words, word)
				word = ""
			}
		} else {
			word += string(r)
		}
	}
	if word != "" {
		words = append(words, word)
	}
	return words
}

type Handler struct {
	queries    *db.Queries
	store      *session.Store
	engine     *recommend.Engine
	tmdbClient *tmdb.Client
}

func New(queries *db.Queries, store *session.Store, engine *recommend.Engine, tmdbClient *tmdb.Client) *Handler {
	return &Handler{queries: queries, store: store, engine: engine, tmdbClient: tmdbClient}
}

func renderPage(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t, ok := pageTemplates[name]
	if !ok {
		slog.Error("template not found", "template", name)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		slog.Error("template render error", "template", name, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func renderPartial(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Try page templates first (for content blocks like "recs-content")
	for _, t := range pageTemplates {
		if tmpl := t.Lookup(name); tmpl != nil {
			if err := tmpl.Execute(w, data); err != nil {
				slog.Error("partial render error", "template", name, "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
	}
	// Fallback to partials
	if err := partialTmpl.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("partial render error", "template", name, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// PartnerInfo holds data for the appbar partner tab
type PartnerInfo struct {
	PartnershipID   int64
	PartnerName     string
	PartnerInitials string
}

// PageData is the base data for full-page renders
type PageData struct {
	ActiveTab    string
	UserName     string
	UserEmail    string
	UserInitials string
	Partners     []PartnerInfo
}

func basePageData(r *http.Request, activeTab string) PageData {
	name := middleware.GetUserName(r.Context())
	email := middleware.GetUserEmail(r.Context())
	ini := makeInitials(name)
	return PageData{
		ActiveTab:    activeTab,
		UserName:     name,
		UserEmail:    email,
		UserInitials: ini,
	}
}

func (h *Handler) basePageDataWithPartners(r *http.Request, activeTab string) PageData {
	pd := basePageData(r, activeTab)
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		return pd
	}

	// Load active partnerships (as owner)
	owned, _ := h.queries.GetActivePartnershipsAsOwner(r.Context(), userID)
	for _, p := range owned {
		pd.Partners = append(pd.Partners, PartnerInfo{
			PartnershipID:   p.ID,
			PartnerName:     p.PartnerName.String,
			PartnerInitials: makeInitials(p.PartnerName.String),
		})
	}

	// Load active partnerships (as partner)
	asPartner, _ := h.queries.GetActivePartnershipsAsPartner(r.Context(), db.NewNullInt64(userID))
	for _, p := range asPartner {
		pd.Partners = append(pd.Partners, PartnerInfo{
			PartnershipID:   p.ID,
			PartnerName:     p.PartnerName.String,
			PartnerInitials: makeInitials(p.PartnerName.String),
		})
	}

	return pd
}

func makeInitials(name string) string {
	ini := ""
	words := splitWords(name)
	for _, w := range words {
		if len(w) > 0 {
			ini += string([]rune(w)[:1])
		}
		if len(ini) >= 2 {
			break
		}
	}
	if ini == "" {
		ini = "?"
	}
	return ini
}
