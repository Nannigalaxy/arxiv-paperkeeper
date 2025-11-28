package arxiv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// ArXiv API base URL
	apiBaseURL = "http://export.arxiv.org/api/query"
	
	// Default timeout for HTTP requests
	defaultTimeout = 30 * time.Second
)

// Client handles communication with the arXiv API
type Client struct {
	httpClient     *http.Client
	rateLimitDelay time.Duration
}

// NewClient creates a new arXiv API client
func NewClient(rateLimitDelay time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		rateLimitDelay: rateLimitDelay,
	}
}

// FetchParams holds parameters for fetching papers
type FetchParams struct {
	Categories []string
	Keywords   []string
	MaxResults int
	SortBy     string // "submittedDate", "lastUpdatedDate", "relevance"
	SortOrder  string // "ascending", "descending"
}

// FetchNew fetches recent papers from arXiv based on the given parameters
func (c *Client) FetchNew(ctx context.Context, params FetchParams) (*Feed, error) {
	// Build search query
	searchQuery := c.buildSearchQuery(params.Categories, params.Keywords)
	
	// Build URL with query parameters
	apiURL, err := c.buildURL(searchQuery, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "ArXiv-PaperKeeper/1.0")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	feed, err := ParseFeed(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Respect rate limiting
	time.Sleep(c.rateLimitDelay)

	return feed, nil
}

// buildSearchQuery constructs the search query string
func (c *Client) buildSearchQuery(categories []string, keywords []string) string {
	var parts []string

	// Add category filters
	if len(categories) > 0 {
		catParts := make([]string, len(categories))
		for i, cat := range categories {
			catParts[i] = fmt.Sprintf("cat:%s", cat)
		}
		if len(catParts) == 1 {
			parts = append(parts, catParts[0])
		} else {
			parts = append(parts, "("+strings.Join(catParts, " OR ")+")")
		}
	}

	// Add keyword filters
	if len(keywords) > 0 {
		kwParts := make([]string, len(keywords))
		for i, kw := range keywords {
			kwParts[i] = fmt.Sprintf("all:%s", kw)
		}
		if len(kwParts) == 1 {
			parts = append(parts, kwParts[0])
		} else {
			parts = append(parts, "("+strings.Join(kwParts, " OR ")+")")
		}
	}

	// Default to all if no filters
	if len(parts) == 0 {
		return "all:*"
	}

	return strings.Join(parts, " AND ")
}

// buildURL constructs the full API URL with parameters
func (c *Client) buildURL(searchQuery string, params FetchParams) (string, error) {
	u, err := url.Parse(apiBaseURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("search_query", searchQuery)
	q.Set("max_results", fmt.Sprintf("%d", params.MaxResults))
	
	// Set sort parameters
	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "submittedDate"
	}
	q.Set("sortBy", sortBy)
	
	sortOrder := params.SortOrder
	if sortOrder == "" {
		sortOrder = "descending"
	}
	q.Set("sortOrder", sortOrder)

	u.RawQuery = q.Encode()
	return u.String(), nil
}

// FetchByIDs fetches specific papers by their arXiv IDs
func (c *Client) FetchByIDs(ctx context.Context, ids []string) (*Feed, error) {
	if len(ids) == 0 {
		return &Feed{}, nil
	}

	// Build ID list query
	idList := strings.Join(ids, ",")
	
	u, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("id_list", idList)
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "ArXiv-PaperKeeper/1.0")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	feed, err := ParseFeed(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// Respect rate limiting
	time.Sleep(c.rateLimitDelay)

	return feed, nil
}
