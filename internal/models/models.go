package models

import "time"

// Paper represents an arXiv paper with all metadata
type Paper struct {
	ID          string    `db:"id"`
	Title       string    `db:"title"`
	Abstract    string    `db:"abstract"`
	Authors     string    `db:"authors"` // JSON array stored as string
	Categories  string    `db:"categories"`
	PublishedAt time.Time `db:"published_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	PDFUrl      string    `db:"pdf_url"`
	ArxivUrl    string    `db:"arxiv_url"`
	CreatedAt   time.Time `db:"created_at"`

	// Fields populated via joins (not in papers table)
	InLibrary bool  `db:"in_library"`
	IsRead    bool  `db:"is_read"`
	Tags      []Tag `db:"-"`
}

// Tag represents a user-defined tag
type Tag struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// LibraryEntry represents a paper saved to the user's library
type LibraryEntry struct {
	PaperID string    `db:"paper_id"`
	IsRead  bool      `db:"is_read"`
	SavedAt time.Time `db:"saved_at"`
}

// PaperTag represents the many-to-many relationship between papers and tags
type PaperTag struct {
	PaperID string `db:"paper_id"`
	TagID   int    `db:"tag_id"`
}

// SearchParams holds parameters for searching and filtering papers
type SearchParams struct {
	Query      string
	Tag        string
	Category   string
	InLibrary  bool
	Page       int
	PageSize   int
	SortBy     string // "published", "title"
	SortOrder  string // "asc", "desc"
}
