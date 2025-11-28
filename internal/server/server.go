package server

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ngx/arxiv-paperkeeper/internal/config"
	"github.com/ngx/arxiv-paperkeeper/internal/db"
)

// Server represents the HTTP server
type Server struct {
	config  *config.Config
	db      *db.DB
	router  *chi.Mux
	handler *Handler
}

// New creates a new HTTP server
func New(cfg *config.Config, database *db.DB) (*Server, error) {
	s := &Server{
		config: cfg,
		db:     database,
		router: chi.NewRouter(),
	}

	// Initialize handler
	handler, err := NewHandler(cfg, database)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}
	s.handler = handler

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Compress(5))
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Serve static files with caching
	staticPath := filepath.Join("web", "static")
	fileServer := http.FileServer(http.Dir(staticPath))
	
	// Wrap file server to add Cache-Control headers
	s.router.Handle("/static/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set long cache duration (1 year) for static assets
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		http.StripPrefix("/static/", fileServer).ServeHTTP(w, r)
	}))

	// HTML routes
	s.router.Get("/", s.handler.HandleIndex)
	s.router.Get("/paper/{id}", s.handler.HandlePaperDetail)
	s.router.Get("/library", s.handler.HandleLibrary)
	s.router.Get("/search", s.handler.HandleSearch)

	// API routes (HTMX endpoints)
	s.router.Post("/library/add/{id}", s.handler.HandleAddToLibrary)
	s.router.Post("/library/remove/{id}", s.handler.HandleRemoveFromLibrary)
	s.router.Post("/library/toggle-read/{id}", s.handler.HandleToggleRead)
	s.router.Post("/tag/add", s.handler.HandleAddTag)
	s.router.Post("/tag/remove", s.handler.HandleRemoveTag)
	
	// Admin routes
	s.router.Post("/admin/refresh", s.handler.HandleRefresh)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := s.config.Address()
	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// Router returns the chi router (useful for testing)
func (s *Server) Router() *chi.Mux {
	return s.router
}
