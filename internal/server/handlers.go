package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	Title              string
	Papers             []models.Paper
	Paper              *models.Paper
	Tags               []models.Tag
	Collections        []models.Collection
	SelectedCollection int // 0 = all/none, -1 = unassigned, >0 = specific ID
	UnassignedCount    int
	CurrentPage        int
	TotalPages         int
	TotalResults       int
	Query              string
	SelectedTag        string
	SelectedCategory   string
	InLibrary          bool
	PaperCount         int
	LibraryCount       int
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

	collections, _ := h.db.GetAllCollections()
	unassignedCount, _ := h.db.GetUnassignedCount()

	paperCount, _ := h.db.GetPaperCount()
	libraryCount, _ := h.db.GetLibraryCount()

	totalPages := (total + h.config.UI.PageSize - 1) / h.config.UI.PageSize

	data := PageData{
		Title:            "ArXiv PaperKeeper",
		Papers:           papers,
		Tags:             tags,
		Collections:      collections,
		UnassignedCount:  unassignedCount,
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

	collections, _ := h.db.GetAllCollections()
	unassignedCount, _ := h.db.GetUnassignedCount()

	paperCount, _ := h.db.GetPaperCount()
	libraryCount, _ := h.db.GetLibraryCount()

	var title string
	if paper != nil {
		title = paper.Title
	} else {
		title = "Paper Not Found"
	}

	data := PageData{
		Title:            title,
		Paper:            paper,
		Tags:             tags,
		Collections:      collections,
		UnassignedCount:  unassignedCount,
		PaperCount:       paperCount,
		LibraryCount:     libraryCount,
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
	category := r.URL.Query().Get("category")
	
	collectionParam := r.URL.Query().Get("collection")
	var collectionID int
	if collectionParam == "unassigned" {
		collectionID = -1
	} else if collectionParam != "" {
		collectionID = getIntParam(r, "collection", 0)
	}

	params := models.SearchParams{
		Query:        query,
		Tag:          tag,
		Category:     category,
		InLibrary:    true,
		CollectionID: collectionID,
		Page:         page,
		PageSize:     h.config.UI.PageSize,
		SortBy:       "published",
		SortOrder:    "desc",
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

	collections, _ := h.db.GetAllCollections()
	unassignedCount, _ := h.db.GetUnassignedCount()

	paperCount, _ := h.db.GetPaperCount()
	libraryCount, _ := h.db.GetLibraryCount()

	totalPages := (total + h.config.UI.PageSize - 1) / h.config.UI.PageSize

	data := PageData{
		Title:              "My Library",
		Papers:             papers,
		Tags:               tags,
		Collections:        collections,
		SelectedCollection: collectionID,
		UnassignedCount:    unassignedCount,
		CurrentPage:        page,
		TotalPages:         totalPages,
		TotalResults:       total,
		Query:              query,
		SelectedTag:        tag,
		SelectedCategory:   category,
		InLibrary:          true,
		PaperCount:         paperCount,
		LibraryCount:       libraryCount,
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
	fmt.Fprintf(w, `<button hx-post="/library/remove/%s" hx-swap="outerHTML" class="btn btn-sm btn-success w-full" title="Saved to Library (Click to Remove)"><i data-lucide="check" class="w-4 h-4"></i> Saved</button><script>lucide.createIcons();</script>`, id)
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
	fmt.Fprintf(w, `<button hx-post="/library/add/%s" hx-swap="outerHTML" class="btn btn-sm btn-outline w-full" title="Save to Library"><i data-lucide="bookmark" class="w-4 h-4"></i> Save</button><script>lucide.createIcons();</script>`, id)
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
		fmt.Fprintf(w, `<button hx-post="/library/toggle-read/%s" hx-swap="outerHTML" class="btn btn-sm btn-success w-full"><i data-lucide="check" class="w-4 h-4"></i> Read</button><script>lucide.createIcons();</script>`, id)
	} else {
		fmt.Fprintf(w, `<button hx-post="/library/toggle-read/%s" hx-swap="outerHTML" class="btn btn-sm btn-outline w-full"><i data-lucide="book-open" class="w-4 h-4"></i> Mark Read</button><script>lucide.createIcons();</script>`, id)
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
	fmt.Fprintf(w, `<span class="text-green-600 dark:text-green-400 flex items-center gap-1.5 justify-center"><i data-lucide="check-circle" class="w-4 h-4"></i> Successfully fetched and stored %d papers</span><script>lucide.createIcons();</script>`, count)
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

// HandleExportLibrary exports saved papers to a CSV file (Title and PDF Link)
func (h *Handler) HandleExportLibrary(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")
	category := r.URL.Query().Get("category")

	collectionParam := r.URL.Query().Get("collection")
	var collectionID int
	if collectionParam == "unassigned" {
		collectionID = -1
	} else if collectionParam != "" {
		collectionID = getIntParam(r, "collection", 0)
	}

	// Query all matching saved papers (no pagination, large page size)
	params := models.SearchParams{
		Query:        query,
		Tag:          tag,
		Category:     category,
		InLibrary:    true,
		CollectionID: collectionID,
		Page:         1,
		PageSize:     1000000, // Very large limit to retrieve all papers
		SortBy:       "published",
		SortOrder:    "desc",
	}

	papers, _, err := h.db.GetPapers(params)
	if err != nil {
		http.Error(w, "Failed to retrieve papers for export", http.StatusInternalServerError)
		log.Printf("Error exporting library: %v", err)
		return
	}

	// Determine file name based on active filter
	fileNamePrefix := "saved_papers"

	if category != "" {
		fileNamePrefix = category
	} else if collectionID > 0 {
		col, err := h.db.GetCollectionByID(collectionID)
		if err == nil && col != nil {
			fileNamePrefix = strings.ToLower(strings.ReplaceAll(col.Name, " ", "_"))
		} else {
			fileNamePrefix = fmt.Sprintf("collection_%d", collectionID)
		}
	} else if collectionID == -1 {
		fileNamePrefix = "unassigned_papers"
	} else if tag != "" {
		fileNamePrefix = strings.ToLower(strings.ReplaceAll(tag, " ", "_"))
	}

	// Sanitize filename prefix
	fileNamePrefix = sanitizeFilename(fileNamePrefix)

	// Format timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.csv", fileNamePrefix, timestamp)

	// Set CSV headers
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// Create CSV writer
	writer := csv.NewWriter(w)

	// Write header row
	if err := writer.Write([]string{"Title", "PDF Link", "Authors", "Categories", "Published Date"}); err != nil {
		log.Printf("Failed to write CSV headers: %v", err)
		return
	}

	// Write rows
	for _, paper := range papers {
		pubDate := paper.PublishedAt.Format("2006-01-02")
		if err := writer.Write([]string{paper.Title, paper.PDFUrl, paper.Authors, paper.Categories, pubDate}); err != nil {
			log.Printf("Failed to write CSV row for paper %s: %v", paper.ID, err)
			continue
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("CSV writer error during flush: %v", err)
	}
}

// sanitizeFilename keeps alphanumeric, underscores and hyphens in the filename
func sanitizeFilename(s string) string {
	var r []rune
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			r = append(r, c)
		} else if c == '.' {
			r = append(r, '_')
		}
	}
	res := string(r)
	if res == "" {
		return "export"
	}
	return res
}

// HandleCreateCollection handles POST /collection/create
func (h *Handler) HandleCreateCollection(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Collection name cannot be empty", http.StatusBadRequest)
		return
	}

	id, err := h.db.CreateCollection(name)
	if err != nil {
		log.Printf("Error creating collection: %v", err)
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s", "type": "error"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/library?collection=%d", id))
	w.WriteHeader(http.StatusOK)
}

// HandleRenameCollection handles POST /collection/rename/{id}
func (h *Handler) HandleRenameCollection(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Collection name cannot be empty", http.StatusBadRequest)
		return
	}

	if err := h.db.RenameCollection(id, name); err != nil {
		log.Printf("Error renaming collection: %v", err)
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s", "type": "error"}}`, err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("HX-Redirect", fmt.Sprintf("/library?collection=%d", id))
	w.WriteHeader(http.StatusOK)
}

// HandleDeleteCollection handles POST /collection/delete/{id}
func (h *Handler) HandleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteCollection(id); err != nil {
		log.Printf("Error deleting collection: %v", err)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to delete collection", "type": "error"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/library")
	w.WriteHeader(http.StatusOK)
}

// HandleAddPaperToCollection handles POST /collection/add-paper
func (h *Handler) HandleAddPaperToCollection(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	paperID := r.FormValue("paper_id")
	collectionIDStr := r.FormValue("collection_id")
	collectionID, err := strconv.Atoi(collectionIDStr)
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	if err := h.db.AddPaperToCollection(collectionID, paperID); err != nil {
		log.Printf("Error adding paper to collection: %v", err)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to add to collection", "type": "error"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"libraryUpdated": true, "showToast": {"message": "Added to collection", "type": "success"}}`)
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// HandleRemovePaperFromCollection handles POST /collection/remove-paper
func (h *Handler) HandleRemovePaperFromCollection(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	paperID := r.FormValue("paper_id")
	collectionIDStr := r.FormValue("collection_id")
	collectionID, err := strconv.Atoi(collectionIDStr)
	if err != nil {
		http.Error(w, "Invalid collection ID", http.StatusBadRequest)
		return
	}

	if err := h.db.RemovePaperFromCollection(collectionID, paperID); err != nil {
		log.Printf("Error removing paper from collection: %v", err)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to remove from collection", "type": "error"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"libraryUpdated": true, "showToast": {"message": "Removed from collection", "type": "info"}}`)
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// HandleMovePaperCollection handles POST /collection/move-paper
func (h *Handler) HandleMovePaperCollection(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	paperID := r.FormValue("paper_id")
	fromIDStr := r.FormValue("from_collection_id")
	toIDStr := r.FormValue("to_collection_id")

	fromID, _ := strconv.Atoi(fromIDStr)
	toID, _ := strconv.Atoi(toIDStr)

	if err := h.db.MovePaperCollection(paperID, fromID, toID); err != nil {
		log.Printf("Error moving paper: %v", err)
		w.Header().Set("HX-Trigger", `{"showToast": {"message": "Failed to move paper", "type": "error"}}`)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Trigger", `{"libraryUpdated": true, "showToast": {"message": "Paper moved successfully", "type": "success"}}`)
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

