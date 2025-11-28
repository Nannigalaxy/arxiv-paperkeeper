package arxiv

import (
	"strings"
	"testing"
	"time"
)

func TestParseFeed(t *testing.T) {
	// Sample Atom feed XML from arXiv
	sampleXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>ArXiv Query: search_query=cat:cs.AI&amp;id_list=&amp;start=0&amp;max_results=1</title>
  <id>http://arxiv.org/api/cHxbiOdZaP56ODnBPIenZhzg5f8</id>
  <updated>2024-01-15T00:00:00-05:00</updated>
  <entry>
    <id>http://arxiv.org/abs/2301.12345v1</id>
    <updated>2023-01-25T12:00:00Z</updated>
    <published>2023-01-25T12:00:00Z</published>
    <title>Test Paper Title</title>
    <summary>This is a test abstract for the paper.</summary>
    <author>
      <name>John Doe</name>
    </author>
    <author>
      <name>Jane Smith</name>
    </author>
    <link href="http://arxiv.org/abs/2301.12345v1" rel="alternate" type="text/html"/>
    <link title="pdf" href="http://arxiv.org/pdf/2301.12345v1" rel="related" type="application/pdf"/>
    <category term="cs.AI" scheme="http://arxiv.org/schemas/atom"/>
    <category term="cs.LG" scheme="http://arxiv.org/schemas/atom"/>
  </entry>
</feed>`

	reader := strings.NewReader(sampleXML)
	feed, err := ParseFeed(reader)

	if err != nil {
		t.Fatalf("ParseFeed failed: %v", err)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(feed.Entries))
	}

	entry := feed.Entries[0]

	// Test entry fields
	if entry.Title != "Test Paper Title" {
		t.Errorf("Expected title 'Test Paper Title', got '%s'", entry.Title)
	}

	if entry.Summary != "This is a test abstract for the paper." {
		t.Errorf("Expected summary 'This is a test abstract for the paper.', got '%s'", entry.Summary)
	}

	if len(entry.Authors) != 2 {
		t.Fatalf("Expected 2 authors, got %d", len(entry.Authors))
	}

	if entry.Authors[0].Name != "John Doe" {
		t.Errorf("Expected first author 'John Doe', got '%s'", entry.Authors[0].Name)
	}

	if len(entry.Categories) != 2 {
		t.Fatalf("Expected 2 categories, got %d", len(entry.Categories))
	}
}

func TestEntryToPaper(t *testing.T) {
	entry := Entry{
		ID:        "http://arxiv.org/abs/2301.12345v1",
		Title:     "  Test   Paper  Title  ",
		Summary:   "  This is a\n  test abstract.  ",
		Published: "2023-01-25T12:00:00Z",
		Updated:   "2023-01-25T12:00:00Z",
		Authors: []Author{
			{Name: "John Doe"},
			{Name: "Jane Smith"},
		},
		Links: []Link{
			{Href: "http://arxiv.org/abs/2301.12345v1", Rel: "alternate"},
			{Href: "http://arxiv.org/pdf/2301.12345v1", Title: "pdf"},
		},
		Categories: []Category{
			{Term: "cs.AI"},
			{Term: "cs.LG"},
		},
	}

	paper, err := entry.ToPaper()
	if err != nil {
		t.Fatalf("ToPaper failed: %v", err)
	}

	// Test arXiv ID extraction
	if paper.ID != "2301.12345" {
		t.Errorf("Expected ID '2301.12345', got '%s'", paper.ID)
	}

	// Test whitespace cleaning
	if paper.Title != "Test Paper Title" {
		t.Errorf("Expected cleaned title 'Test Paper Title', got '%s'", paper.Title)
	}

	if paper.Abstract != "This is a test abstract." {
		t.Errorf("Expected cleaned abstract, got '%s'", paper.Abstract)
	}

	// Test authors
	if paper.Authors != "John Doe, Jane Smith" {
		t.Errorf("Expected authors 'John Doe, Jane Smith', got '%s'", paper.Authors)
	}

	// Test categories
	if paper.Categories != "cs.AI, cs.LG" {
		t.Errorf("Expected categories 'cs.AI, cs.LG', got '%s'", paper.Categories)
	}

	// Test URLs
	if paper.PDFUrl != "http://arxiv.org/pdf/2301.12345v1" {
		t.Errorf("Expected PDF URL 'http://arxiv.org/pdf/2301.12345v1', got '%s'", paper.PDFUrl)
	}

	if paper.ArxivUrl != "http://arxiv.org/abs/2301.12345v1" {
		t.Errorf("Expected arXiv URL 'http://arxiv.org/abs/2301.12345v1', got '%s'", paper.ArxivUrl)
	}

	// Test timestamp parsing
	expectedTime := time.Date(2023, 1, 25, 12, 0, 0, 0, time.UTC)
	if !paper.PublishedAt.Equal(expectedTime) {
		t.Errorf("Expected published time %v, got %v", expectedTime, paper.PublishedAt)
	}
}

func TestExtractArxivID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"http://arxiv.org/abs/2301.12345v1", "2301.12345"},
		{"http://arxiv.org/abs/2301.12345v2", "2301.12345"},
		{"http://arxiv.org/abs/1234.56789v1", "1234.56789"},
		{"2301.12345v1", "2301.12345"},
		{"invalid", ""},
	}

	for _, test := range tests {
		result := extractArxivID(test.input)
		if result != test.expected {
			t.Errorf("extractArxivID(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  world  ", "hello world"},
		{"hello\n\nworld", "hello world"},
		{"  multiple   spaces  ", "multiple spaces"},
		{"normal text", "normal text"},
	}

	for _, test := range tests {
		result := cleanText(test.input)
		if result != test.expected {
			t.Errorf("cleanText(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestFeedToPapers(t *testing.T) {
	feed := &Feed{
		Entries: []Entry{
			{
				ID:        "http://arxiv.org/abs/2301.12345v1",
				Title:     "Paper 1",
				Summary:   "Abstract 1",
				Published: "2023-01-25T12:00:00Z",
				Updated:   "2023-01-25T12:00:00Z",
				Authors:   []Author{{Name: "Author 1"}},
				Links: []Link{
					{Href: "http://arxiv.org/abs/2301.12345v1", Rel: "alternate"},
					{Href: "http://arxiv.org/pdf/2301.12345v1", Title: "pdf"},
				},
				Categories: []Category{{Term: "cs.AI"}},
			},
			{
				ID:        "http://arxiv.org/abs/2301.67890v1",
				Title:     "Paper 2",
				Summary:   "Abstract 2",
				Published: "2023-01-26T12:00:00Z",
				Updated:   "2023-01-26T12:00:00Z",
				Authors:   []Author{{Name: "Author 2"}},
				Links: []Link{
					{Href: "http://arxiv.org/abs/2301.67890v1", Rel: "alternate"},
					{Href: "http://arxiv.org/pdf/2301.67890v1", Title: "pdf"},
				},
				Categories: []Category{{Term: "cs.LG"}},
			},
		},
	}

	papers, err := feed.ToPapers()
	if err != nil {
		t.Fatalf("ToPapers failed: %v", err)
	}

	if len(papers) != 2 {
		t.Fatalf("Expected 2 papers, got %d", len(papers))
	}

	if papers[0].ID != "2301.12345" {
		t.Errorf("Expected first paper ID '2301.12345', got '%s'", papers[0].ID)
	}

	if papers[1].ID != "2301.67890" {
		t.Errorf("Expected second paper ID '2301.67890', got '%s'", papers[1].ID)
	}
}
