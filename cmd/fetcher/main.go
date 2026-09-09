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
	"wind_analysis/internal/client"
	"wind_analysis/internal/database"
	"wind_analysis/pipeline"

	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1) DB initialisieren
	err := godotenv.Load()
	if err != nil {
		log.Println("keine .env gefunden")
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		user,
		password,
		host,
		port,
		dbname,
	)

	db, err := database.New(ctx, connString)
	if err != nil {
		log.Fatalf("DB-Verbindung fehlgeschlagen: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("Schema-Init fehlgeschlagen: %v", err)
	}

	// 2) Config laden
	configPath := flag.String("config", "config.yaml", "Pfad zur Konfigurationsdatei")
	flag.Parse()

	cfg, err := pipeline.Load(*configPath)
	if err != nil {
		log.Fatalf("Fehler beim Laden der Konfiguration: %v", err)
	}
	log.Printf("📋 Konfiguration geladen: %d Standorte, Worker: %d, Rate-Limit: %d RPS",
		len(cfg.LocationList), cfg.Pipeline.Concurrency, cfg.Pipeline.RateLimitRPS)

	// 3) API-Client initialisieren
	apiClient := client.NewSmartClient()

	// 4) Rate Limiter initialisieren
	rateLimiter := client.NewRateLimiter(cfg.Pipeline.RateLimitRPS)
	defer rateLimiter.Stop()

	// 5) Pipeline starten
	engine := pipeline.NewEngine(cfg, db, apiClient, rateLimiter)
	log.Println("⚙️ Pipeline-Prozess gestartet...")
	startTime := time.Now()
	if err := pipeline.Run(engine, ctx); err != nil {
		// Prüfen, ob der Abbruch manuell durch den Benutzer erfolgte
		if ctx.Err() == context.Canceled {
			log.Println("⚠️ Pipeline wurde durch den Benutzer manuell abgebrochen (Graceful Shutdown executed).")
		} else {
			log.Fatalf("❌ Pipeline abgebrochen mit Fehler: %v", err)
		}
	} else {
		log.Printf("🎉 Erfolgreich beendet! Gesamtlaufzeit: %v", time.Since(startTime).Round(time.Second))
	}
}
