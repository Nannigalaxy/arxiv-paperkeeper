package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/ngx/arxiv-paperkeeper/internal/models"
)

// UpsertPaper inserts or updates a paper in the database
func (db *DB) UpsertPaper(paper *models.Paper) error {
	query := `
		INSERT INTO papers (id, title, abstract, authors, categories, published_at, updated_at, pdf_url, arxiv_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			abstract = excluded.abstract,
			authors = excluded.authors,
			categories = excluded.categories,
			published_at = excluded.published_at,
			updated_at = excluded.updated_at,
			pdf_url = excluded.pdf_url,
			arxiv_url = excluded.arxiv_url
	`
	_, err := db.Exec(query,
		paper.ID, paper.Title, paper.Abstract, paper.Authors,
		paper.Categories, paper.PublishedAt, paper.UpdatedAt,
		paper.PDFUrl, paper.ArxivUrl,
	)
	return err
}

// GetPapers retrieves papers with optional filtering, searching, and pagination
func (db *DB) GetPapers(params models.SearchParams) ([]models.Paper, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}

	if params.Query != "" {
		conditions = append(conditions, "(p.title LIKE ? OR p.abstract LIKE ? OR p.authors LIKE ?)")
		searchTerm := "%" + params.Query + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if params.Category != "" {
		conditions = append(conditions, "p.categories LIKE ?")
		args = append(args, "%"+params.Category+"%")
	}

	if params.InLibrary {
		conditions = append(conditions, "l.paper_id IS NOT NULL")
	}

	if params.Tag != "" {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM paper_tags pt
			JOIN tags t ON pt.tag_id = t.id
			WHERE pt.paper_id = p.id AND t.name = ?
		)`)
		args = append(args, params.Tag)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total results
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM papers p
		LEFT JOIN library l ON p.id = l.paper_id
		%s
	`, whereClause)

	var total int
	if err := db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to count papers: %w", err)
	}

	// Build ORDER BY clause
	sortBy := "p.published_at"
	if params.SortBy == "title" {
		sortBy = "p.title"
	}
	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Calculate offset
	offset := (params.Page - 1) * params.PageSize
	if offset < 0 {
		offset = 0
	}

	// Fetch papers
	query := fmt.Sprintf(`
		SELECT DISTINCT
			p.id, p.title, p.abstract, p.authors, p.categories, 
			p.published_at, p.updated_at, p.pdf_url, p.arxiv_url,
			l.paper_id IS NOT NULL AS in_library,
			COALESCE(l.is_read, 0) AS is_read
		FROM papers p
		LEFT JOIN library l ON p.id = l.paper_id
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortBy, sortOrder)

	args = append(args, params.PageSize, offset)

	var papers []models.Paper
	if err := db.Select(&papers, query, args...); err != nil {
		return nil, 0, fmt.Errorf("failed to fetch papers: %w", err)
	}

	// Fetch tags for each paper
	for i := range papers {
		tags, err := db.GetPaperTags(papers[i].ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to fetch tags for paper %s: %w", papers[i].ID, err)
		}
		papers[i].Tags = tags
	}

	return papers, total, nil
}

// GetPaperByID retrieves a single paper by ID
func (db *DB) GetPaperByID(id string) (*models.Paper, error) {
	query := `
		SELECT
			p.*,
			CASE WHEN l.paper_id IS NOT NULL THEN 1 ELSE 0 END as in_library,
			COALESCE(l.is_read, 0) as is_read
		FROM papers p
		LEFT JOIN library l ON p.id = l.paper_id
		WHERE p.id = ?
	`

	var paper models.Paper
	if err := db.Get(&paper, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("paper not found: %s", id)
		}
		return nil, fmt.Errorf("failed to fetch paper: %w", err)
	}

	// Fetch tags
	tags, err := db.GetPaperTags(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}
	paper.Tags = tags

	return &paper, nil
}

// SaveToLibrary adds a paper to the user's library
func (db *DB) SaveToLibrary(paperID string) error {
	query := `INSERT INTO library (paper_id) VALUES (?) ON CONFLICT(paper_id) DO NOTHING`
	_, err := db.Exec(query, paperID)
	return err
}

// RemoveFromLibrary removes a paper from the user's library
func (db *DB) RemoveFromLibrary(paperID string) error {
	query := `DELETE FROM library WHERE paper_id = ?`
	_, err := db.Exec(query, paperID)
	return err
}

// ToggleRead toggles the read status of a paper in the library
func (db *DB) ToggleRead(paperID string) error {
	query := `UPDATE library SET is_read = NOT is_read WHERE paper_id = ?`
	_, err := db.Exec(query, paperID)
	return err
}

// CreateTag creates a new tag or returns existing tag ID
func (db *DB) CreateTag(name string) (int, error) {
	// Try to get existing tag
	var tag models.Tag
	err := db.Get(&tag, "SELECT * FROM tags WHERE name = ?", name)
	if err == nil {
		return tag.ID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to check for existing tag: %w", err)
	}

	// Create new tag
	result, err := db.Exec("INSERT INTO tags (name) VALUES (?)", name)
	if err != nil {
		return 0, fmt.Errorf("failed to create tag: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get tag ID: %w", err)
	}

	return int(id), nil
}

// TagPaper associates a tag with a paper
func (db *DB) TagPaper(paperID string, tagID int) error {
	query := `INSERT INTO paper_tags (paper_id, tag_id) VALUES (?, ?) ON CONFLICT DO NOTHING`
	_, err := db.Exec(query, paperID, tagID)
	return err
}

// UntagPaper removes a tag from a paper
func (db *DB) UntagPaper(paperID string, tagID int) error {
	query := `DELETE FROM paper_tags WHERE paper_id = ? AND tag_id = ?`
	_, err := db.Exec(query, paperID, tagID)
	return err
}

// GetPaperTags retrieves all tags for a paper
func (db *DB) GetPaperTags(paperID string) ([]models.Tag, error) {
	query := `
		SELECT t.* FROM tags t
		JOIN paper_tags pt ON t.id = pt.tag_id
		WHERE pt.paper_id = ?
		ORDER BY t.name
	`

	var tags []models.Tag
	if err := db.Select(&tags, query, paperID); err != nil {
		return nil, err
	}

	if tags == nil {
		tags = []models.Tag{}
	}

	return tags, nil
}

// GetAllTags retrieves all tags with paper counts
func (db *DB) GetAllTags() ([]models.Tag, error) {
	query := `SELECT * FROM tags ORDER BY name`

	var tags []models.Tag
	if err := db.Select(&tags, query); err != nil {
		return nil, err
	}

	if tags == nil {
		tags = []models.Tag{}
	}

	return tags, nil
}

// GetPaperCount returns the total number of papers
func (db *DB) GetPaperCount() (int, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM papers")
	return count, err
}

// GetLibraryCount returns the number of papers in the library
func (db *DB) GetLibraryCount() (int, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM library")
	return count, err
}
