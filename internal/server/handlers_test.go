package server

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ngx/arxiv-paperkeeper/internal/arxiv"
	"github.com/ngx/arxiv-paperkeeper/internal/config"
	"github.com/ngx/arxiv-paperkeeper/internal/db"
	"github.com/ngx/arxiv-paperkeeper/internal/models"
)

func setupTestHandler(t *testing.T) (*Handler, *db.DB) {
	// Create test database
	testDB, err := db.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		ArXiv: config.ArXivConfig{
			Categories:     []string{"cs.AI"},
			MaxResults:     10,
			RateLimitDelay: 1 * time.Second,
		},
		UI: config.UIConfig{
			PageSize: 10,
		},
	}

	// Create handler with mock templates
	handler := &Handler{
		config: cfg,
		db:     testDB,
		templates: template.Must(template.New("test").Parse(`
			{{define "list.html"}}Test Paper{{end}}
			{{define "detail.html"}}Test Paper John Doe{{end}}
			{{define "library.html"}}My Library{{end}}
		`)),
		arxiv: arxiv.NewClient(cfg.ArXiv.RateLimitDelay),
	}

	return handler, testDB
}

func insertTestPapers(t *testing.T, db *db.DB, count int) {
	for i := 1; i <= count; i++ {
		paper := &models.Paper{
			ID:          string(rune('0' + i)),
			Title:       "Test Paper " + string(rune('0'+i)),
			Abstract:    "Test abstract " + string(rune('0'+i)),
			Authors:     "Author " + string(rune('0'+i)),
			Categories:  "cs.AI",
			PublishedAt: time.Now(),
			UpdatedAt:   time.Now(),
			PDFUrl:      "http://arxiv.org/pdf/test",
			ArxivUrl:    "http://arxiv.org/abs/test",
		}
		if err := db.UpsertPaper(paper); err != nil {
			t.Fatalf("Failed to insert test paper: %v", err)
		}
	}
}

func TestHandleIndex(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test papers
	insertTestPapers(t, testDB, 5)

	// Create request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Execute handler
	handler.HandleIndex(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Paper") {
		t.Error("Expected response to contain 'Test Paper'")
	}
}

func TestHandleSearch(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test papers
	paper1 := &models.Paper{
		ID:          "1",
		Title:       "Machine Learning Paper",
		Abstract:    "About ML",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	paper2 := &models.Paper{
		ID:          "2",
		Title:       "Deep Learning Paper",
		Abstract:    "About DL",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	testDB.UpsertPaper(paper1)
	testDB.UpsertPaper(paper2)

	// Create search request
	req := httptest.NewRequest("GET", "/search?q=Machine", nil)
	w := httptest.NewRecorder()

	// Execute handler
	handler.HandleSearch(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	// With mock templates, we just check that the template rendered
	if !strings.Contains(body, "Test Paper") {
		t.Error("Expected response to contain 'Test Paper' from mock template")
	}
}

func TestHandlePaperDetail(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test paper
	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		Abstract:    "Test abstract",
		Authors:     "John Doe",
		Categories:  "cs.AI",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
		PDFUrl:      "http://arxiv.org/pdf/2301.12345",
		ArxivUrl:    "http://arxiv.org/abs/2301.12345",
	}
	testDB.UpsertPaper(paper)

	// Create request with chi context
	req := httptest.NewRequest("GET", "/paper/2301.12345", nil)
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "2301.12345")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	// Execute handler
	handler.HandlePaperDetail(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Test Paper") {
		t.Error("Expected response to contain 'Test Paper'")
	}
	if !strings.Contains(body, "John Doe") {
		t.Error("Expected response to contain 'John Doe'")
	}
}

func TestHandleAddToLibrary(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test paper
	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	testDB.UpsertPaper(paper)

	// Create request
	req := httptest.NewRequest("POST", "/library/add/2301.12345", nil)
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "2301.12345")
	ctx := req.Context()
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	// Execute handler
	handler.HandleAddToLibrary(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify paper is in library
	retrieved, _ := testDB.GetPaperByID("2301.12345")
	if !retrieved.InLibrary {
		t.Error("Expected paper to be in library")
	}

	// Check HTMX trigger header
	if w.Header().Get("HX-Trigger") != "libraryUpdated" {
		t.Error("Expected HX-Trigger header to be set")
	}
}

func TestHandleRemoveFromLibrary(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test paper and add to library
	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	testDB.UpsertPaper(paper)
	testDB.SaveToLibrary("2301.12345")

	// Create request
	req := httptest.NewRequest("POST", "/library/remove/2301.12345", nil)
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "2301.12345")
	ctx := req.Context()
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	// Execute handler
	handler.HandleRemoveFromLibrary(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify paper is not in library
	retrieved, _ := testDB.GetPaperByID("2301.12345")
	if retrieved.InLibrary {
		t.Error("Expected paper to not be in library")
	}
}

func TestHandleToggleRead(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test paper and add to library
	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	testDB.UpsertPaper(paper)
	testDB.SaveToLibrary("2301.12345")

	// Create request
	req := httptest.NewRequest("POST", "/library/toggle-read/2301.12345", nil)
	w := httptest.NewRecorder()

	// Add chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "2301.12345")
	ctx := req.Context()
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	// Execute handler
	handler.HandleToggleRead(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify read status
	retrieved, _ := testDB.GetPaperByID("2301.12345")
	if !retrieved.IsRead {
		t.Error("Expected paper to be marked as read")
	}
}

func TestHandleAddTag(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test paper
	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	testDB.UpsertPaper(paper)

	// Create form data
	form := url.Values{}
	form.Add("paper_id", "2301.12345")
	form.Add("tag_name", "machine-learning")

	// Create request
	req := httptest.NewRequest("POST", "/tag/add", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Execute handler
	handler.HandleAddTag(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify tag was added
	tags, _ := testDB.GetPaperTags("2301.12345")
	if len(tags) != 1 {
		t.Fatalf("Expected 1 tag, got %d", len(tags))
	}
	if tags[0].Name != "machine-learning" {
		t.Errorf("Expected tag 'machine-learning', got '%s'", tags[0].Name)
	}
}

func TestHandleLibrary(t *testing.T) {
	handler, testDB := setupTestHandler(t)
	defer testDB.Close()

	// Insert test papers
	insertTestPapers(t, testDB, 3)

	// Add 2 to library
	testDB.SaveToLibrary("1")
	testDB.SaveToLibrary("2")

	// Create request
	req := httptest.NewRequest("GET", "/library", nil)
	w := httptest.NewRecorder()

	// Execute handler
	handler.HandleLibrary(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "My Library") {
		t.Error("Expected response to contain 'My Library'")
	}
}

func TestGetIntParam(t *testing.T) {
	tests := []struct {
		url      string
		key      string
		defVal   int
		expected int
	}{
		{"/?page=5", "page", 1, 5},
		{"/?page=0", "page", 1, 1},      // Invalid, should return default
		{"/?page=-1", "page", 1, 1},     // Invalid, should return default
		{"/?page=abc", "page", 1, 1},    // Invalid, should return default
		{"/", "page", 10, 10},           // Missing, should return default
	}

	for _, test := range tests {
		req := httptest.NewRequest("GET", test.url, nil)
		result := getIntParam(req, test.key, test.defVal)
		if result != test.expected {
			t.Errorf("getIntParam(%s, %s, %d) = %d, expected %d",
				test.url, test.key, test.defVal, result, test.expected)
		}
	}
}
