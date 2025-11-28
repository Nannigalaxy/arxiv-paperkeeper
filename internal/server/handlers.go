package server

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ngx/arxiv-paperkeeper/internal/arxiv"
	"github.com/ngx/arxiv-paperkeeper/internal/config"
	"github.com/ngx/arxiv-paperkeeper/internal/db"
	"github.com/ngx/arxiv-paperkeeper/internal/models"
)

// Handler handles HTTP requests
type Handler struct {
	config    *config.Config
	db        *db.DB
	templates *template.Template
	arxiv     *arxiv.Client
}

// NewHandler creates a new handler
func NewHandler(cfg *config.Config, database *db.DB) (*Handler, error) {
	// Parse templates with helper functions
	tmpl, err := NewTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Create arXiv client
	arxivClient := arxiv.NewClient(cfg.ArXiv.RateLimitDelay)

	return &Handler{
		config:    cfg,
		db:        database,
		templates: tmpl,
		arxiv:     arxivClient,
	}, nil
}

// PageData holds common data for all pages
type PageData struct {
	Title            string
	Papers           []models.Paper
	Paper            *models.Paper
	Tags             []models.Tag
	CurrentPage      int
	TotalPages       int
	TotalResults     int
	Query            string
	SelectedTag      string
	SelectedCategory string
	InLibrary        bool
	PaperCount       int
	LibraryCount     int
}

// HandleIndex renders the main paper list page
func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	page := getIntParam(r, "page", 1)
	query := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")
	category := r.URL.Query().Get("category")

	params := models.SearchParams{
		Query:     query,
		Tag:       tag,
		Category:  category,
		InLibrary: false,
		Page:      page,
		PageSize:  h.config.UI.PageSize,
		SortBy:    "published",
		SortOrder: "desc",
	}

	papers, total, err := h.db.GetPapers(params)
	if err != nil {
		http.Error(w, "Failed to fetch papers", http.StatusInternalServerError)
		log.Printf("Error fetching papers: %v", err)
		return
	}

	tags, err := h.db.GetAllTags()
	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		tags = []models.Tag{}
	}

	paperCount, _ := h.db.GetPaperCount()
	libraryCount, _ := h.db.GetLibraryCount()

	totalPages := (total + h.config.UI.PageSize - 1) / h.config.UI.PageSize

	data := PageData{
		Title:            "ArXiv PaperKeeper",
		Papers:           papers,
		Tags:             tags,
		CurrentPage:      page,
		TotalPages:       totalPages,
		TotalResults:     total,
		Query:            query,
		SelectedTag:      tag,
		SelectedCategory: category,
		PaperCount:       paperCount,
		LibraryCount:     libraryCount,
	}

	if err := h.templates.ExecuteTemplate(w, "list.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Error rendering template: %v", err)
	}
}

// HandlePaperDetail renders the paper detail page
func (h *Handler) HandlePaperDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	paper, err := h.db.GetPaperByID(id)
	if err != nil {
		log.Printf("Error fetching paper %s: %v", id, err)
		// Don't return error - render template with nil paper
		// Template will show "Paper not found" message
	}

	tags, err := h.db.GetAllTags()
	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		tags = []models.Tag{}
	}

	paperCount, _ := h.db.GetPaperCount()
	libraryCount, _ := h.db.GetLibraryCount()

	var title string
	if paper != nil {
		title = paper.Title
	} else {
		title = "Paper Not Found"
	}

	data := PageData{
		Title:        title,
		Paper:        paper,
		Tags:         tags,
		PaperCount:   paperCount,
		LibraryCount: libraryCount,
	}

	if err := h.templates.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Error rendering template: %v", err)
	}
}

// HandleLibrary renders the user's library page
func (h *Handler) HandleLibrary(w http.ResponseWriter, r *http.Request) {
	page := getIntParam(r, "page", 1)
	query := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")

	params := models.SearchParams{
		Query:     query,
		Tag:       tag,
		InLibrary: true,
		Page:      page,
		PageSize:  h.config.UI.PageSize,
		SortBy:    "published",
		SortOrder: "desc",
	}

	papers, total, err := h.db.GetPapers(params)
	if err != nil {
		http.Error(w, "Failed to fetch library", http.StatusInternalServerError)
		log.Printf("Error fetching library: %v", err)
		return
	}

	tags, err := h.db.GetAllTags()
	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		tags = []models.Tag{}
	}

	paperCount, _ := h.db.GetPaperCount()
	libraryCount, _ := h.db.GetLibraryCount()

	totalPages := (total + h.config.UI.PageSize - 1) / h.config.UI.PageSize

	data := PageData{
		Title:        "My Library",
		Papers:       papers,
		Tags:         tags,
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalResults: total,
		Query:        query,
		SelectedTag:  tag,
		InLibrary:    true,
		PaperCount:   paperCount,
		LibraryCount: libraryCount,
	}

	if err := h.templates.ExecuteTemplate(w, "library.html", data); err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		log.Printf("Error rendering template: %v", err)
	}
}

// HandleSearch handles search requests (same as index but with query)
func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	h.HandleIndex(w, r)
}

// HandleAddToLibrary adds a paper to the library (HTMX endpoint)
func (h *Handler) HandleAddToLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.db.SaveToLibrary(id); err != nil {
		http.Error(w, "Failed to add to library", http.StatusInternalServerError)
		log.Printf("Error adding to library: %v", err)
		return
	}

	w.Header().Set("HX-Trigger", `{"libraryUpdated": true, "showToast": {"message": "Saved to library", "type": "success"}}`)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<button hx-post="/library/remove/%s" hx-swap="outerHTML" class="btn btn-success flex-1 md:flex-none md:w-full" title="Saved to Library (Click to Remove)"><i data-lucide="check" class="w-4 h-4"></i></button><script>lucide.createIcons();</script>`, id)
}

// HandleRemoveFromLibrary removes a paper from the library (HTMX endpoint)
func (h *Handler) HandleRemoveFromLibrary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.db.RemoveFromLibrary(id); err != nil {
		http.Error(w, "Failed to remove from library", http.StatusInternalServerError)
		log.Printf("Error removing from library: %v", err)
		return
	}

	w.Header().Set("HX-Trigger", `{"libraryUpdated": true, "showToast": {"message": "Removed from library", "type": "info"}}`)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<button hx-post="/library/add/%s" hx-swap="outerHTML" class="btn btn-outline flex-1 md:flex-none md:w-full" title="Save to Library"><i data-lucide="bookmark" class="w-4 h-4"></i></button><script>lucide.createIcons();</script>`, id)
}

// HandleToggleRead toggles the read status (HTMX endpoint)
func (h *Handler) HandleToggleRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.db.ToggleRead(id); err != nil {
		http.Error(w, "Failed to toggle read status", http.StatusInternalServerError)
		log.Printf("Error toggling read status: %v", err)
		return
	}

	// Fetch updated paper to get current read status
	paper, err := h.db.GetPaperByID(id)
	if err != nil {
		http.Error(w, "Failed to fetch paper", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if paper.IsRead {
		fmt.Fprintf(w, `<button hx-post="/library/toggle-read/%s" hx-swap="outerHTML" class="btn btn-sm btn-success">✓ Read</button>`, id)
	} else {
		fmt.Fprintf(w, `<button hx-post="/library/toggle-read/%s" hx-swap="outerHTML" class="btn btn-sm btn-outline">Mark as Read</button>`, id)
	}
}

// HandleAddTag adds a tag to a paper (HTMX endpoint)
func (h *Handler) HandleAddTag(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	paperID := r.FormValue("paper_id")
	tagName := strings.TrimSpace(r.FormValue("tag_name"))

	if paperID == "" || tagName == "" {
		http.Error(w, "Missing paper_id or tag_name", http.StatusBadRequest)
		return
	}

	// Create or get tag
	tagID, err := h.db.CreateTag(tagName)
	if err != nil {
		http.Error(w, "Failed to create tag", http.StatusInternalServerError)
		log.Printf("Error creating tag: %v", err)
		return
	}

	// Associate tag with paper
	if err := h.db.TagPaper(paperID, tagID); err != nil {
		http.Error(w, "Failed to tag paper", http.StatusInternalServerError)
		log.Printf("Error tagging paper: %v", err)
		return
	}

	// Return updated tag list
	tags, err := h.db.GetPaperTags(paperID)
	if err != nil {
		http.Error(w, "Failed to fetch tags", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	for _, tag := range tags {
		fmt.Fprintf(w, `<span class="tag">%s <button hx-post="/tag/remove" hx-vals='{"paper_id":"%s","tag_id":%d}' hx-target="#tags-%s" hx-swap="innerHTML" class="tag-remove">×</button></span> `, tag.Name, paperID, tag.ID, paperID)
	}
}

// HandleRemoveTag removes a tag from a paper (HTMX endpoint)
func (h *Handler) HandleRemoveTag(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	paperID := r.FormValue("paper_id")
	tagIDStr := r.FormValue("tag_id")

	tagID, err := strconv.Atoi(tagIDStr)
	if err != nil {
		http.Error(w, "Invalid tag_id", http.StatusBadRequest)
		return
	}

	if err := h.db.UntagPaper(paperID, tagID); err != nil {
		http.Error(w, "Failed to remove tag", http.StatusInternalServerError)
		log.Printf("Error removing tag: %v", err)
		return
	}

	// Return updated tag list
	tags, err := h.db.GetPaperTags(paperID)
	if err != nil {
		http.Error(w, "Failed to fetch tags", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	for _, tag := range tags {
		fmt.Fprintf(w, `<span class="tag">%s <button hx-post="/tag/remove" hx-vals='{"paper_id":"%s","tag_id":%d}' hx-target="#tags-%s" hx-swap="innerHTML" class="tag-remove">×</button></span> `, tag.Name, paperID, tag.ID, paperID)
	}
}

// HandleRefresh manually triggers a fetch of new papers
func (h *Handler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	params := arxiv.FetchParams{
		Categories: h.config.ArXiv.Categories,
		Keywords:   h.config.ArXiv.Keywords,
		MaxResults: h.config.ArXiv.MaxResults,
		SortBy:     "submittedDate",
		SortOrder:  "descending",
	}

	feed, err := h.arxiv.FetchNew(ctx, params)
	if err != nil {
		http.Error(w, "Failed to fetch papers", http.StatusInternalServerError)
		log.Printf("Error fetching papers: %v", err)
		return
	}

	papers, err := feed.ToPapers()
	if err != nil {
		http.Error(w, "Failed to parse papers", http.StatusInternalServerError)
		log.Printf("Error parsing papers: %v", err)
		return
	}

	// Insert papers into database
	count := 0
	for _, paper := range papers {
		if err := h.db.UpsertPaper(paper); err != nil {
			log.Printf("Error inserting paper %s: %v", paper.ID, err)
			continue
		}
		count++
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<span class="text-green-600 dark:text-green-400">✓ Successfully fetched and stored %d papers</span>`, count)
}

// getIntParam extracts an integer parameter from the URL query string
func getIntParam(r *http.Request, key string, defaultValue int) int {
	valueStr := r.URL.Query().Get(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	if value < 1 {
		return defaultValue
	}

	return value
}
