package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ngx/arxiv-paperkeeper/internal/arxiv"
	"github.com/ngx/arxiv-paperkeeper/internal/config"
	"github.com/ngx/arxiv-paperkeeper/internal/db"
	"github.com/ngx/arxiv-paperkeeper/internal/server"
)

const (
	defaultConfigPath = "config.yaml"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", defaultConfigPath, "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	database, err := db.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Parse command
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"server"} // Default to server command
	}

	command := args[0]

	switch command {
	case "server":
		runServer(cfg, database)
	case "fetch":
		runFetch(cfg, database)
	case "migrate":
		fmt.Println("Database migrations completed successfully")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Available commands: server, fetch, migrate\n")
		os.Exit(1)
	}
}

// runServer starts the HTTP server with background scheduler
func runServer(cfg *config.Config, database *db.DB) {
	// Create server
	srv, err := server.New(cfg, database)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start background scheduler
	stopScheduler := startScheduler(cfg, database)
	defer stopScheduler()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", cfg.Address())
		errChan <- srv.Start()
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		log.Fatalf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
	}
}

// runFetch manually fetches new papers from arXiv
func runFetch(cfg *config.Config, database *db.DB) {
	ctx := context.Background()
	client := arxiv.NewClient(cfg.ArXiv.RateLimitDelay)

	params := arxiv.FetchParams{
		Categories: cfg.ArXiv.Categories,
		Keywords:   cfg.ArXiv.Keywords,
		MaxResults: cfg.ArXiv.MaxResults,
		SortBy:     "submittedDate",
		SortOrder:  "descending",
	}

	log.Printf("Fetching papers from arXiv...")
	log.Printf("Categories: %v", params.Categories)
	log.Printf("Max results: %d", params.MaxResults)

	feed, err := client.FetchNew(ctx, params)
	if err != nil {
		log.Fatalf("Failed to fetch papers: %v", err)
	}

	papers, err := feed.ToPapers()
	if err != nil {
		log.Fatalf("Failed to parse papers: %v", err)
	}

	log.Printf("Fetched %d papers, inserting into database...", len(papers))

	count := 0
	for _, paper := range papers {
		if err := database.UpsertPaper(paper); err != nil {
			log.Printf("Error inserting paper %s: %v", paper.ID, err)
			continue
		}
		count++
	}

	log.Printf("Successfully stored %d papers", count)
}

// startScheduler starts a background goroutine that fetches papers periodically
func startScheduler(cfg *config.Config, database *db.DB) func() {
	ticker := time.NewTicker(cfg.ArXiv.FetchInterval)
	stopChan := make(chan struct{})

	go func() {
		// Run initial fetch after a short delay
		time.Sleep(10 * time.Second)
		fetchPapers(cfg, database)

		// Then run on schedule
		for {
			select {
			case <-ticker.C:
				fetchPapers(cfg, database)
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	// Return stop function
	return func() {
		close(stopChan)
	}
}

// fetchPapers fetches and stores papers from arXiv
func fetchPapers(cfg *config.Config, database *db.DB) {
	ctx := context.Background()
	client := arxiv.NewClient(cfg.ArXiv.RateLimitDelay)

	params := arxiv.FetchParams{
		Categories: cfg.ArXiv.Categories,
		Keywords:   cfg.ArXiv.Keywords,
		MaxResults: cfg.ArXiv.MaxResults,
		SortBy:     "submittedDate",
		SortOrder:  "descending",
	}

	log.Printf("Scheduled fetch: fetching papers from arXiv...")

	feed, err := client.FetchNew(ctx, params)
	if err != nil {
		log.Printf("Error fetching papers: %v", err)
		return
	}

	papers, err := feed.ToPapers()
	if err != nil {
		log.Printf("Error parsing papers: %v", err)
		return
	}

	count := 0
	for _, paper := range papers {
		if err := database.UpsertPaper(paper); err != nil {
			log.Printf("Error inserting paper %s: %v", paper.ID, err)
			continue
		}
		count++
	}

	log.Printf("Scheduled fetch: stored %d papers", count)
}
