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

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/validation"
	"wind_analysis/models"
)

func main() {
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
		// Hier können nun generisch die zu testenden Verteilungen definiert werden
		Fitters: []distribution.Fitter{
			distribution.WeibullFitter{},
			distribution.LogNormalFitter{},
			distribution.GammaFitter{},
		},
	}

	fmt.Println("🚀 Starte Standardlauf: Einzel-Analysen & Standortvergleich")

	// 6. Einzelstandort-Analysen durchführen
	successfulLocations := make([]models.Location, 0, len(config.LocationList))

	for _, location := range config.LocationList {
		fmt.Printf("\n📍 Starte Analyse für Standort: %s\n", location.Name)
		locConfig := analysisConfig
		locConfig.LocationName = location.Name

		if err := validation.RunSingleLocationAnalysis(ctx, db, locConfig); err != nil {
			fmt.Printf("⚠️ Standort %s übersprungen oder unvollständig: %v\n", location.Name, err)
			continue
		}

		successfulLocations = append(successfulLocations, location)
	}

	// 7. Standortübergreifender Vergleich (nur wenn mindestens 2 Standorte Daten geliefert haben)
	if len(successfulLocations) >= 2 {
		fmt.Println("\n📊 Starte standortübergreifenden Vergleich...")
		if err := validation.RunLocationComparison(ctx, db, analysisConfig, successfulLocations); err != nil {
			fmt.Printf("⚠️ Fehler beim Standortvergleich: %v\n", err)
		}
	} else {
		fmt.Println("\nℹ️ Standortvergleich übersprungen (weniger als 2 erfolgreiche Standorte).")
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
