package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"wind_analysis/internal/database"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	db     *database.DB
	router *http.ServeMux
}

func New(db *database.DB) *Server {
	s := &Server{db: db}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router = http.NewServeMux()

	// Statische Dateien (HTML, JS, CSS)
	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Fehler beim Einbinden der statischen Dateien: %v", err)
	}
	s.router.Handle("/", http.FileServer(http.FS(static)))

	// API Endpoints
	s.router.HandleFunc("GET /api/locations", s.handleLocations)
	s.router.HandleFunc("POST /api/timeseries", s.handleTimeSeries)
	s.router.HandleFunc("POST /api/distributions", s.handleDistributions)
	s.router.HandleFunc("POST /api/validation", s.handleValidation)
}

func Start(db *database.DB, port string) error {
	s := New(db)
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server läuft auf %s", addr)
	return http.ListenAndServe(addr, s.router)
}
