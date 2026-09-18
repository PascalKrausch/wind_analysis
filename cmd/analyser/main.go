package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"wind_analysis/internal/analysis/validation"

	"wind_analysis/models"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Datenbank initialisieren
	db, err := setupDatabase(ctx)
	if err != nil {
		log.Fatalf("Datenbank-Setup fehlgeschlagen: %v", err)
	}
	defer db.Close()

	// Konfiguration laden
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Fehler beim Laden der Konfiguration: %v", err)
	}

	// Ausgabeverzeichnis erstellen
	outputDir := "./output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Fehler beim Erstellen des Ausgabeverzeichnisses: %v", err)
	}

	// Analyse-Konfiguration
	analysisConfig := validation.Config{
		StartTime: "2022-01-01",
		EndTime:   time.Now().Format("2006-01-02"),
		OutputDir: outputDir,
	}

	// Analyse für alle Standorte durchführen
	fmt.Println("🚀 Starte Standardlauf: alle Standorte + Standortvergleich")
	for _, location := range config.LocationList {
		fmt.Printf("\n📍 Standort: %s\n", location.Name)
		analysisConfig.LocationName = location.Name

		if err := validation.RunSingleLocationAnalysis(ctx, db, analysisConfig); err != nil {
			fmt.Printf("⚠️ Standort %s übersprungen: %v\n", location.Name, err)
		}
	}

	// Standortvergleich durchführen
	if err := validation.RunLocationComparison(ctx, db, analysisConfig, config.LocationList); err != nil {
		log.Fatalf("Fehler bei Standortvergleich: %v", err)
	}

	fmt.Println("✅ Analyse erfolgreich abgeschlossen!")
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
