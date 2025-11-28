package arxiv

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/ngx/arxiv-paperkeeper/internal/models"
)

// Feed represents the Atom feed returned by arXiv API
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	ID      string   `xml:"id"`
	Updated string   `xml:"updated"`
	Entries []Entry  `xml:"entry"`
}

// Entry represents a single paper in the Atom feed
type Entry struct {
	ID        string   `xml:"id"`
	Title     string   `xml:"title"`
	Summary   string   `xml:"summary"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Authors   []Author `xml:"author"`
	Links     []Link   `xml:"link"`
	Categories []Category `xml:"category"`
}

// Author represents a paper author
type Author struct {
	Name string `xml:"name"`
}

// Link represents a link to the paper
type Link struct {
	Href  string `xml:"href,attr"`
	Rel   string `xml:"rel,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
}

// Category represents an arXiv category
type Category struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
}

var (
	// Regex to extract arXiv ID from URL
	arxivIDRegex = regexp.MustCompile(`(\d{4}\.\d{4,5})(v\d+)?$`)
	
	// Regex to clean whitespace
	whitespaceRegex = regexp.MustCompile(`\s+`)
)

// ParseFeed parses an Atom feed from arXiv API
func ParseFeed(r io.Reader) (*Feed, error) {
	var feed Feed
	decoder := xml.NewDecoder(r)
	
	if err := decoder.Decode(&feed); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	return &feed, nil
}

// ToPaper converts an Entry to a models.Paper
func (e *Entry) ToPaper() (*models.Paper, error) {
	// Extract arXiv ID
	arxivID := extractArxivID(e.ID)
	if arxivID == "" {
		return nil, fmt.Errorf("failed to extract arXiv ID from: %s", e.ID)
	}

	// Parse timestamps
	publishedAt, err := parseTime(e.Published)
	if err != nil {
		return nil, fmt.Errorf("failed to parse published date: %w", err)
	}

	updatedAt, err := parseTime(e.Updated)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated date: %w", err)
	}

	// Extract author names
	authors := make([]string, len(e.Authors))
	for i, author := range e.Authors {
		authors[i] = strings.TrimSpace(author.Name)
	}

	// Extract categories
	categories := make([]string, len(e.Categories))
	for i, cat := range e.Categories {
		categories[i] = cat.Term
	}

	// Find PDF and arXiv URLs
	var pdfURL, arxivURL string
	for _, link := range e.Links {
		if link.Title == "pdf" {
			pdfURL = link.Href
		} else if link.Rel == "alternate" {
			arxivURL = link.Href
		}
	}

	// Clean title and abstract
	title := cleanText(e.Title)
	abstract := cleanText(e.Summary)

	paper := &models.Paper{
		ID:          arxivID,
		Title:       title,
		Abstract:    abstract,
		Authors:     strings.Join(authors, ", "),
		Categories:  strings.Join(categories, ", "),
		PublishedAt: publishedAt,
		UpdatedAt:   updatedAt,
		PDFUrl:      pdfURL,
		ArxivUrl:    arxivURL,
	}

	return paper, nil
}

// ToPapers converts all entries in a feed to papers
func (f *Feed) ToPapers() ([]*models.Paper, error) {
	papers := make([]*models.Paper, 0, len(f.Entries))
	
	for i, entry := range f.Entries {
		paper, err := entry.ToPaper()
		if err != nil {
			// Log error but continue processing other entries
			fmt.Printf("Warning: failed to convert entry %d: %v\n", i, err)
			continue
		}
		papers = append(papers, paper)
	}

	return papers, nil
}

// extractArxivID extracts the arXiv ID from a URL or ID string
func extractArxivID(idStr string) string {
	// Try to extract from URL
	matches := arxivIDRegex.FindStringSubmatch(idStr)
	if len(matches) > 1 {
		return matches[1] // Return without version number
	}
	return ""
}

// parseTime parses an ISO 8601 timestamp
func parseTime(timeStr string) (time.Time, error) {
	// Try multiple formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
	}

	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse time: %s", timeStr)
}

// cleanText removes extra whitespace and newlines
func cleanText(text string) string {
	// Replace multiple whitespace with single space
	text = whitespaceRegex.ReplaceAllString(text, " ")
	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)
	return text
}
