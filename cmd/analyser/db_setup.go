package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"wind_analysis/internal/database"
)

// setupDatabase initialisiert die Datenbankverbindung
func setupDatabase(ctx context.Context) (*database.DB, error) {
	// .env Datei laden (optional)
	if err := godotenv.Load(); err != nil {
		log.Println("keine .env gefunden")
	}

	// Umgebungsvariablen lesen
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// Connection String bauen
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		user,
		password,
		host,
		port,
		dbname,
	)

	// Datenbankverbindung herstellen
	db, err := database.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("DB-Verbindung fehlgeschlagen: %w", err)
	}

	return db, nil
}