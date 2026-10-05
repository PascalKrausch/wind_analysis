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

	"gopkg.in/yaml.v3"

	"wind_analysis/internal/analysis/validation"
	"wind_analysis/internal/server"
	"wind_analysis/models"
)

func main() {
	// Prüfe ob serve Subcommand aufgerufen wird
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServer()
		return
	}

	// 1. Graceful Shutdown einrichten
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Datenbank initialisieren
	db, err := setupDatabase(ctx)
	if err != nil {
		log.Fatalf("Datenbank-Setup fehlgeschlagen: %v", err)
	}
	defer db.Close()

	// 3. Konfiguration laden
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Fehler beim Laden der Konfiguration: %v", err)
	}

	// 4. Ausgabeverzeichnis erstellen
	outputDir := "./output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Fehler beim Erstellen des Ausgabeverzeichnisses: %v", err)
	}

	// 5. Global gültige Analyse-Konfiguration
	analysisConfig := validation.Config{
		StartTime: "2022-01-01",
		EndTime:   time.Now().Format("2006-01-02"),
		OutputDir: outputDir,
		Fitters:   validation.DefaultFitters(),
	}

	fmt.Println("🚀 Starte Standardlauf: Einzel-Analysen & Standortvergleich")

	// 6. Einzelstandort-Analysen durchführen
	for _, location := range config.LocationList {
		fmt.Printf("\n📍 Starte Analyse für Standort: %s\n", location.Name)
		locConfig := analysisConfig
		locConfig.LocationName = location.Name

		if err := validation.RunSingleLocationAnalysis(ctx, db, locConfig); err != nil {
			fmt.Printf("⚠️ Standort %s übersprungen oder unvollständig: %v\n", location.Name, err)
			continue
		}

	}

	// 7. Vergleich und Master-Dashboard für alle konfigurierten Standorte
	if len(config.LocationList) > 0 {
		fmt.Println("\n📊 Starte Standortvergleich und Master-Dashboard...")
		if err := validation.RunLocationComparison(ctx, db, analysisConfig, config.LocationList); err != nil {
			fmt.Printf("⚠️ Fehler beim Standortvergleich: %v\n", err)
		}
	} else {
		fmt.Println("\nℹ️ Standortvergleich und Master-Dashboard übersprungen (keine Standorte konfiguriert).")
	}

	fmt.Println("\n✅ Analyse erfolgreich abgeschlossen!")
}

func loadConfig(configPath string) (*models.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func runServer() {
	// Flags parsen
	port := flag.String("port", "8080", "Port für den Server")
	flag.Parse()

	// Datenbank initialisieren
	ctx := context.Background()
	db, err := setupDatabase(ctx)
	if err != nil {
		log.Fatalf("Datenbank-Setup fehlgeschlagen: %v", err)
	}
	defer db.Close()

	// Server starten
	log.Printf("🚀 Starte Server auf http://localhost:%s", *port)
	if err := server.Start(db, *port); err != nil {
		log.Fatalf("Server-Fehler: %v", err)
	}
}
