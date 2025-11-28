package db

import (
	"os"
	"testing"
	"time"

	"github.com/ngx/arxiv-paperkeeper/internal/models"
)

func setupTestDB(t *testing.T) *DB {
	// Create temporary database
	tmpfile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()

	db, err := New(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Clean up function
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpfile.Name())
	})

	return db
}

func TestUpsertPaper(t *testing.T) {
	db := setupTestDB(t)

	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		Abstract:    "Test abstract",
		Authors:     "John Doe, Jane Smith",
		Categories:  "cs.AI, cs.LG",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
		PDFUrl:      "http://arxiv.org/pdf/2301.12345",
		ArxivUrl:    "http://arxiv.org/abs/2301.12345",
	}

	// Insert paper
	err := db.UpsertPaper(paper)
	if err != nil {
		t.Fatalf("UpsertPaper failed: %v", err)
	}

	// Retrieve paper
	retrieved, err := db.GetPaperByID("2301.12345")
	if err != nil {
		t.Fatalf("GetPaperByID failed: %v", err)
	}

	if retrieved.Title != paper.Title {
		t.Errorf("Expected title '%s', got '%s'", paper.Title, retrieved.Title)
	}

	// Update paper
	paper.Title = "Updated Title"
	err = db.UpsertPaper(paper)
	if err != nil {
		t.Fatalf("UpsertPaper (update) failed: %v", err)
	}

	// Retrieve updated paper
	updated, err := db.GetPaperByID("2301.12345")
	if err != nil {
		t.Fatalf("GetPaperByID (after update) failed: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got '%s'", updated.Title)
	}
}

func TestGetPapers(t *testing.T) {
	db := setupTestDB(t)

	// Insert test papers
	papers := []*models.Paper{
		{
			ID:          "2301.00001",
			Title:       "Machine Learning Paper",
			Abstract:    "About machine learning",
			Authors:     "Alice",
			Categories:  "cs.LG",
			PublishedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * time.Hour),
		},
		{
			ID:          "2301.00002",
			Title:       "AI Paper",
			Abstract:    "About artificial intelligence",
			Authors:     "Bob",
			Categories:  "cs.AI",
			PublishedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		},
		{
			ID:          "2301.00003",
			Title:       "Deep Learning Paper",
			Abstract:    "About deep learning",
			Authors:     "Charlie",
			Categories:  "cs.LG",
			PublishedAt: time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, p := range papers {
		if err := db.UpsertPaper(p); err != nil {
			t.Fatalf("Failed to insert paper: %v", err)
		}
	}

	// Test: Get all papers
	params := models.SearchParams{
		Page:     1,
		PageSize: 10,
		SortBy:   "published",
		SortOrder: "desc",
	}

	results, total, err := db.GetPapers(params)
	if err != nil {
		t.Fatalf("GetPapers failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected 3 total papers, got %d", total)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Test: Search by query
	params.Query = "machine"
	results, total, err = db.GetPapers(params)
	if err != nil {
		t.Fatalf("GetPapers (search) failed: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected 1 result for 'machine', got %d", total)
	}

	// Test: Filter by category
	params.Query = ""
	params.Category = "cs.LG"
	results, total, err = db.GetPapers(params)
	if err != nil {
		t.Fatalf("GetPapers (category filter) failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 results for cs.LG, got %d", total)
	}

	// Test: Pagination
	params.Category = ""
	params.PageSize = 2
	results, total, err = db.GetPapers(params)
	if err != nil {
		t.Fatalf("GetPapers (pagination) failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results per page, got %d", len(results))
	}
}

func TestLibraryOperations(t *testing.T) {
	db := setupTestDB(t)

	// Insert test paper
	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	db.UpsertPaper(paper)

	// Test: Save to library
	err := db.SaveToLibrary("2301.12345")
	if err != nil {
		t.Fatalf("SaveToLibrary failed: %v", err)
	}

	// Verify paper is in library
	retrieved, err := db.GetPaperByID("2301.12345")
	if err != nil {
		t.Fatalf("GetPaperByID failed: %v", err)
	}

	if !retrieved.InLibrary {
		t.Error("Expected paper to be in library")
	}

	// Test: Toggle read status
	err = db.ToggleRead("2301.12345")
	if err != nil {
		t.Fatalf("ToggleRead failed: %v", err)
	}

	retrieved, _ = db.GetPaperByID("2301.12345")
	if !retrieved.IsRead {
		t.Error("Expected paper to be marked as read")
	}

	// Toggle again
	db.ToggleRead("2301.12345")
	retrieved, _ = db.GetPaperByID("2301.12345")
	if retrieved.IsRead {
		t.Error("Expected paper to be marked as unread")
	}

	// Test: Remove from library
	err = db.RemoveFromLibrary("2301.12345")
	if err != nil {
		t.Fatalf("RemoveFromLibrary failed: %v", err)
	}

	retrieved, _ = db.GetPaperByID("2301.12345")
	if retrieved.InLibrary {
		t.Error("Expected paper to be removed from library")
	}
}

func TestTagOperations(t *testing.T) {
	db := setupTestDB(t)

	// Insert test paper
	paper := &models.Paper{
		ID:          "2301.12345",
		Title:       "Test Paper",
		PublishedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}
	db.UpsertPaper(paper)

	// Test: Create tag
	tagID, err := db.CreateTag("machine-learning")
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	if tagID == 0 {
		t.Error("Expected non-zero tag ID")
	}

	// Test: Create duplicate tag (should return existing ID)
	tagID2, err := db.CreateTag("machine-learning")
	if err != nil {
		t.Fatalf("CreateTag (duplicate) failed: %v", err)
	}

	if tagID != tagID2 {
		t.Errorf("Expected same tag ID for duplicate, got %d and %d", tagID, tagID2)
	}

	// Test: Tag paper
	err = db.TagPaper("2301.12345", tagID)
	if err != nil {
		t.Fatalf("TagPaper failed: %v", err)
	}

	// Verify tag is associated
	tags, err := db.GetPaperTags("2301.12345")
	if err != nil {
		t.Fatalf("GetPaperTags failed: %v", err)
	}

	if len(tags) != 1 {
		t.Fatalf("Expected 1 tag, got %d", len(tags))
	}

	if tags[0].Name != "machine-learning" {
		t.Errorf("Expected tag 'machine-learning', got '%s'", tags[0].Name)
	}

	// Test: Add another tag
	tagID3, _ := db.CreateTag("deep-learning")
	db.TagPaper("2301.12345", tagID3)

	tags, _ = db.GetPaperTags("2301.12345")
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	// Test: Untag paper
	err = db.UntagPaper("2301.12345", tagID)
	if err != nil {
		t.Fatalf("UntagPaper failed: %v", err)
	}

	tags, _ = db.GetPaperTags("2301.12345")
	if len(tags) != 1 {
		t.Errorf("Expected 1 tag after removal, got %d", len(tags))
	}
}

func TestGetPaperCount(t *testing.T) {
	db := setupTestDB(t)

	// Initially should be 0
	count, err := db.GetPaperCount()
	if err != nil {
		t.Fatalf("GetPaperCount failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 papers, got %d", count)
	}

	// Insert papers
	for i := 1; i <= 5; i++ {
		paper := &models.Paper{
			ID:          string(rune('0' + i)),
			Title:       "Paper",
			PublishedAt: time.Now(),
			UpdatedAt:   time.Now(),
		}
		db.UpsertPaper(paper)
	}

	count, _ = db.GetPaperCount()
	if count != 5 {
		t.Errorf("Expected 5 papers, got %d", count)
	}
}

func TestGetLibraryCount(t *testing.T) {
	db := setupTestDB(t)

	// Insert papers
	for i := 1; i <= 3; i++ {
		paper := &models.Paper{
			ID:          string(rune('0' + i)),
			Title:       "Paper",
			PublishedAt: time.Now(),
			UpdatedAt:   time.Now(),
		}
		db.UpsertPaper(paper)
	}

	// Add 2 to library
	db.SaveToLibrary("1")
	db.SaveToLibrary("2")

	count, err := db.GetLibraryCount()
	if err != nil {
		t.Fatalf("GetLibraryCount failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 papers in library, got %d", count)
	}
}
