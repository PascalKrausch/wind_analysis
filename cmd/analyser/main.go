package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"wind_analysis/internal/analysis/interpolation"
	"wind_analysis/internal/analysis/visualization"
	"wind_analysis/internal/database"
	"wind_analysis/models"
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

	// Feste Defaults (keine CLI-Flags)
	startTime := "2022-01-01"
	endTime := time.Now().Format("2006-01-02")
	outputDir := "./output"

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Fehler beim Erstellen des Ausgabeverzeichnisses: %v", err)
	}

	if err := runAllPlotsDefault(ctx, db, startTime, endTime, outputDir); err != nil {
		log.Fatalf("Fehler bei der Gesamtausführung: %v", err)
	}

	fmt.Println("✅ Analyse erfolgreich abgeschlossen!")
}

func runAllPlotsDefault(ctx context.Context, db *database.DB, startTime, endTime, outputDir string) error {
	config, err := loadConfig("config.yaml")
	if err != nil {
		return fmt.Errorf("Fehler beim Laden der Konfiguration: %w", err)
	}

	fmt.Println("🚀 Starte Standardlauf: alle Standorte + Standortvergleich")
	for _, location := range config.LocationList {
		fmt.Printf("\n📍 Standort: %s\n", location.Name)
		if err := runSingleLocationAnalysis(ctx, db, location.Name, startTime, endTime, outputDir); err != nil {
			fmt.Printf("⚠️ Standort %s übersprungen: %v\n", location.Name, err)
		}
	}

	if err := runLocationComparison(ctx, db, startTime, endTime, outputDir); err != nil {
		return err
	}
	return nil
}

func runSingleLocationAnalysis(ctx context.Context, db *database.DB, locationName, startTime, endTime, outputDir string) error {
	fmt.Printf("📊 Lade Winddaten für %s von %s bis %s...\n", locationName, startTime, endTime)

	// Winddaten laden
	records, err := db.LoadWindData(ctx, locationName, startTime, endTime)
	if err != nil {
		return fmt.Errorf("Fehler beim Laden der Winddaten: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("keine Daten für den angegebenen Zeitraum gefunden")
	}

	fmt.Printf("✅ %d Datensätze geladen\n", len(records))

	// Hellmann-Exponenten berechnen
	fmt.Println("🔬 Berechne Hellmann-Exponenten...")
	exponents := interpolation.CalculateHellmannExponentsForDataset(records)
	fmt.Printf("✅ %d gültige Hellmann-Exponenten berechnet\n", len(exponents))

	// Timeline-Visualisierung erstellen
	timelinePath := filepath.Join(outputDir, fmt.Sprintf("hellmann_timeline_%s.html", locationName))
	if err := visualization.PlotHellmannExponentTimeline(exponents, timelinePath); err != nil {
		return fmt.Errorf("Fehler beim Erstellen der Timeline: %w", err)
	}
	fmt.Printf("📈 Timeline gespeichert: %s\n", timelinePath)

	// Validierungsmetriken als Tabelle nach Höhe und Ort
	fmt.Println("🔍 Validiere Power-Law Modell...")

	recordsFrom2022 := interpolation.FilterRecordsFromYear(records, 2022)
	if len(recordsFrom2022) == 0 {
		fmt.Println("⚠️ Keine Daten ab 2022 für Validierung verfügbar")
		return nil
	}

	validationByLocation := interpolation.ValidatePowerLawModelByLocation(recordsFrom2022, exponents)
	if len(validationByLocation) == 0 {
		fmt.Println("⚠️ Keine Validierungsergebnisse verfügbar")
		return nil
	}

	metricsPath := filepath.Join(outputDir, fmt.Sprintf("validation_metrics_by_location_%s.html", sanitizeForFilename(locationName)))
	if err := visualization.PlotValidationMetricsByHeightAndLocation(validationByLocation, metricsPath); err != nil {
		return fmt.Errorf("Fehler beim Erstellen der Metrik-Tabelle: %w", err)
	}
	fmt.Printf("📋 Validierungsmetriken (Tabelle) gespeichert: %s\n", metricsPath)

	descriptivePath := filepath.Join(outputDir, fmt.Sprintf("validation_descriptive_stats_%s.html", sanitizeForFilename(locationName)))
	if err := visualization.PlotValidationDescriptiveStatsTable(validationByLocation, descriptivePath); err != nil {
		return fmt.Errorf("Fehler beim Erstellen der deskriptiven Statistik-Tabelle: %w", err)
	}
	fmt.Printf("📋 Deskriptive Statistik gespeichert: %s\n", descriptivePath)

	errorShapePath := filepath.Join(outputDir, fmt.Sprintf("validation_error_shape_%s.html", sanitizeForFilename(locationName)))
	if err := visualization.PlotErrorDistributionShapeByHeight(validationByLocation, locationName, errorShapePath); err != nil {
		fmt.Printf("⚠️ Fehler beim Erstellen des Error-Shape-Charts für %s: %v\n", locationName, err)
	} else {
		fmt.Printf("📊 Error-Shape-Chart gespeichert: %s\n", errorShapePath)
	}

	return nil
}

func runLocationComparison(ctx context.Context, db *database.DB, startTime, endTime, outputDir string) error {
	// Konfiguration laden für Standortliste
	config, err := loadConfig("config.yaml")
	if err != nil {
		return fmt.Errorf("Fehler beim Laden der Konfiguration: %w", err)
	}

	var allExponents []interpolation.HellmannExponentResult
	var allRecordsFrom2022 []models.WindRecord

	fmt.Println("🔍 Sammle Daten für Standortvergleich...")
	for _, location := range config.LocationList {
		fmt.Printf("  - Lade Daten für %s...\n", location.Name)

		records, err := db.LoadWindData(ctx, location.Name, startTime, endTime)
		if err != nil {
			fmt.Printf("  ⚠️ Fehler beim Laden der Daten für %s: %v\n", location.Name, err)
			continue
		}
		if len(records) == 0 {
			fmt.Printf("  ⚠️ Keine Daten für %s gefunden\n", location.Name)
			continue
		}

		exponents := interpolation.CalculateHellmannExponentsForDataset(records)
		allExponents = append(allExponents, exponents...)
		allRecordsFrom2022 = append(allRecordsFrom2022, interpolation.FilterRecordsFromYear(records, 2022)...)

		fmt.Printf("  ✅ %d Exponenten für %s\n", len(exponents), location.Name)
	}

	if len(allExponents) == 0 {
		return fmt.Errorf("keine Exponenten für Standortvergleich gesammelt")
	}

	// Standortvergleich als Chart (bleibt)
	comparisonPath := filepath.Join(outputDir, "location_comparison.html")
	if err := visualization.PlotMultiLocationComparison(allExponents, comparisonPath); err != nil {
		return fmt.Errorf("Fehler beim Erstellen des Standortvergleichs: %w", err)
	}
	fmt.Printf("📈 Standortvergleich gespeichert: %s\n", comparisonPath)

	// Validierungsmetriken als Tabelle nach Höhe und Ort (gesamt)
	if len(allRecordsFrom2022) > 0 {
		validationByLocation := interpolation.ValidatePowerLawModelByLocation(allRecordsFrom2022, allExponents)
		if len(validationByLocation) > 0 {
			metricsPath := filepath.Join(outputDir, "validation_metrics_by_height_and_location.html")
			if err := visualization.PlotValidationMetricsByHeightAndLocation(validationByLocation, metricsPath); err != nil {
				return fmt.Errorf("Fehler beim Erstellen der globalen Metrik-Tabelle: %w", err)
			}
			fmt.Printf("📋 Globale Validierungsmetriken gespeichert: %s\n", metricsPath)

			descriptivePath := filepath.Join(outputDir, "validation_descriptive_stats_by_height_and_location.html")
			if err := visualization.PlotValidationDescriptiveStatsTable(validationByLocation, descriptivePath); err != nil {
				return fmt.Errorf("Fehler beim Erstellen der globalen deskriptiven Statistik-Tabelle: %w", err)
			}
			fmt.Printf("📋 Globale deskriptive Statistik gespeichert: %s\n", descriptivePath)

			locations := make([]string, 0, len(validationByLocation))
			for loc := range validationByLocation {
				locations = append(locations, loc)
			}
			sort.Strings(locations)
			for _, loc := range locations {
				fmt.Printf("  ✅ Metriken vorhanden für: %s\n", loc)

				errorShapePath := filepath.Join(outputDir, fmt.Sprintf("validation_error_shape_%s.html", sanitizeForFilename(loc)))
				if err := visualization.PlotErrorDistributionShapeByHeight(validationByLocation, loc, errorShapePath); err != nil {
					fmt.Printf("  ⚠️ Error-Shape-Chart für %s übersprungen: %v\n", loc, err)
					continue
				}
				fmt.Printf("  📊 Error-Shape-Chart gespeichert: %s\n", errorShapePath)
			}
		}
	}

	return nil
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

func sanitizeForFilename(name string) string {
	s := strings.TrimSpace(strings.ToLower(name))
	s = strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	).Replace(s)
	if s == "" {
		return "unbekannt"
	}
	return s
}
