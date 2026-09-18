package validation

import (
	"context"
	"fmt"
	"sort"

	"wind_analysis/internal/analysis/interpolation"
	"wind_analysis/internal/analysis/visualization"
	"wind_analysis/internal/analysis/weibull"
	"wind_analysis/internal/database"
	"wind_analysis/internal/utils"
	"wind_analysis/models"
)

// Config enthält die Konfiguration für die Validierungsanalyse
type Config struct {
	StartTime    string
	EndTime      string
	OutputDir    string
	LocationName string // Optional für einzelne Standort-Analyse
}

// RunSingleLocationAnalysis führt die Analyse für einen einzelnen Standort durch
func RunSingleLocationAnalysis(ctx context.Context, db *database.DB, config Config) error {
	fmt.Printf("📊 Lade Winddaten für %s von %s bis %s...\n", config.LocationName, config.StartTime, config.EndTime)

	// Winddaten laden
	records, err := db.LoadWindData(ctx, config.LocationName, config.StartTime, config.EndTime)
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
	timelinePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("hellmann_timeline_%s.html", utils.SanitizeForFilename(config.LocationName)))
	if err := visualization.PlotHellmannExponentTimeline(exponents, timelinePath); err != nil {
		return fmt.Errorf("Fehler beim Erstellen der Timeline: %w", err)
	}
	fmt.Printf("📈 Timeline gespeichert: %s\n", timelinePath)

	// Validierungsmetriken
	runValidationMetrics(ctx, db, config, records, exponents)

	// Weibull-Analyse
	runWeibullAnalysis(config, records)

	return nil
}

// RunLocationComparison führt den Vergleich zwischen allen Standorten durch
func RunLocationComparison(ctx context.Context, db *database.DB, config Config, locationList []models.Location) error {
	var allExponents []interpolation.HellmannExponentResult
	var allRecordsFrom2022 []models.WindRecord
	var weibullInputs []visualization.WeibullPlotInput
	var weibullResults []models.WeibullAnalysisResult

	fmt.Println("🔍 Sammle Daten für Standortvergleich...")
	for _, location := range locationList {
		fmt.Printf("  - Lade Daten für %s...\n", location.Name)

		records, err := db.LoadWindData(ctx, location.Name, config.StartTime, config.EndTime)
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

		// Weibull-Analyse
		weibullRecords := interpolation.FilterRecordsFromYear(records, 2022)
		if len(weibullRecords) == 0 {
			weibullRecords = records
		}

		inputs, results, err := weibull.BuildPlotsForLocation(location.Name, weibullRecords)
		if err == nil {
			weibullInputs = append(weibullInputs, inputs...)
			weibullResults = append(weibullResults, results...)
		}

		fmt.Printf("  ✅ %d Exponenten für %s\n", len(exponents), location.Name)
	}

	if len(allExponents) == 0 {
		return fmt.Errorf("keine Exponenten für Standortvergleich gesammelt")
	}

	// Standortvergleich als Chart
	comparisonPath := utils.BuildOutputPath(config.OutputDir, "location_comparison.html")
	if err := visualization.PlotMultiLocationComparison(allExponents, comparisonPath); err != nil {
		return fmt.Errorf("Fehler beim Erstellen des Standortvergleichs: %w", err)
	}
	fmt.Printf("📈 Standortvergleich gespeichert: %s\n", comparisonPath)

	// Globale Validierungsmetriken
	runGlobalValidationMetrics(config, allRecordsFrom2022, allExponents)

	// Weibull-Vergleich
	if len(weibullInputs) > 0 {
		weibullComparisonPath := utils.BuildOutputPath(config.OutputDir, "weibull_location_comparison.html")
		comparisons := weibull.BuildComparisons(weibullResults)
		if err := visualization.PlotWeibullComparisonOverview(weibullInputs, comparisons, weibullComparisonPath); err != nil {
			return fmt.Errorf("Fehler beim Erstellen des Weibull-Vergleichs: %w", err)
		}
		fmt.Printf("🌬️ Weibull-Vergleich gespeichert: %s\n", weibullComparisonPath)
	}

	return nil
}

// runValidationMetrics führt die Validierungsmetrik-Berechnung durch
func runValidationMetrics(ctx context.Context, db *database.DB, config Config, records []models.WindRecord, exponents []interpolation.HellmannExponentResult) {
	fmt.Println("🔍 Validiere Power-Law Modell...")

	recordsFrom2022 := interpolation.FilterRecordsFromYear(records, 2022)
	if len(recordsFrom2022) == 0 {
		fmt.Println("⚠️ Keine Daten ab 2022 für Validierung verfügbar")
		return
	}

	validationByLocation := interpolation.ValidatePowerLawModelByLocation(recordsFrom2022, exponents)
	if len(validationByLocation) == 0 {
		fmt.Println("⚠️ Keine Validierungsergebnisse verfügbar")
		return
	}

	metricsPath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_metrics_by_location_%s.html", utils.SanitizeForFilename(config.LocationName)))
	if err := visualization.PlotValidationMetricsByHeightAndLocation(validationByLocation, metricsPath); err != nil {
		fmt.Printf("⚠️ Fehler beim Erstellen der Metrik-Tabelle: %v\n", err)
		return
	}
	fmt.Printf("📋 Validierungsmetriken (Tabelle) gespeichert: %s\n", metricsPath)

	descriptivePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_descriptive_stats_%s.html", utils.SanitizeForFilename(config.LocationName)))
	if err := visualization.PlotValidationDescriptiveStatsTable(validationByLocation, descriptivePath); err != nil {
		fmt.Printf("⚠️ Fehler beim Erstellen der deskriptiven Statistik-Tabelle: %v\n", err)
		return
	}
	fmt.Printf("📋 Deskriptive Statistik gespeichert: %s\n", descriptivePath)

	errorShapePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_error_shape_%s.html", utils.SanitizeForFilename(config.LocationName)))
	if err := visualization.PlotErrorDistributionShapeByHeight(validationByLocation, config.LocationName, errorShapePath); err != nil {
		fmt.Printf("⚠️ Fehler beim Erstellen des Error-Shape-Charts für %s: %v\n", config.LocationName, err)
	} else {
		fmt.Printf("📊 Error-Shape-Chart gespeichert: %s\n", errorShapePath)
	}
}

// runGlobalValidationMetrics führt die globale Validierung durch
func runGlobalValidationMetrics(config Config, allRecordsFrom2022 []models.WindRecord, allExponents []interpolation.HellmannExponentResult) {
	if len(allRecordsFrom2022) == 0 {
		return
	}

	validationByLocation := interpolation.ValidatePowerLawModelByLocation(allRecordsFrom2022, allExponents)
	if len(validationByLocation) == 0 {
		return
	}

	metricsPath := utils.BuildOutputPath(config.OutputDir, "validation_metrics_by_height_and_location.html")
	if err := visualization.PlotValidationMetricsByHeightAndLocation(validationByLocation, metricsPath); err != nil {
		fmt.Printf("⚠️ Fehler beim Erstellen der globalen Metrik-Tabelle: %v\n", err)
		return
	}
	fmt.Printf("📋 Globale Validierungsmetriken gespeichert: %s\n", metricsPath)

	descriptivePath := utils.BuildOutputPath(config.OutputDir, "validation_descriptive_stats_by_height_and_location.html")
	if err := visualization.PlotValidationDescriptiveStatsTable(validationByLocation, descriptivePath); err != nil {
		fmt.Printf("⚠️ Fehler beim Erstellen der globalen deskriptiven Statistik-Tabelle: %v\n", err)
		return
	}
	fmt.Printf("📋 Globale deskriptive Statistik gespeichert: %s\n", descriptivePath)

	locations := make([]string, 0, len(validationByLocation))
	for loc := range validationByLocation {
		locations = append(locations, loc)
	}
	sort.Strings(locations)

	for _, loc := range locations {
		fmt.Printf("  ✅ Metriken vorhanden für: %s\n", loc)

		errorShapePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_error_shape_%s.html", utils.SanitizeForFilename(loc)))
		if err := visualization.PlotErrorDistributionShapeByHeight(validationByLocation, loc, errorShapePath); err != nil {
			fmt.Printf("  ⚠️ Error-Shape-Chart für %s übersprungen: %v\n", loc, err)
			continue
		}
		fmt.Printf("  📊 Error-Shape-Chart gespeichert: %s\n", errorShapePath)
	}
}

// runWeibullAnalysis führt die Weibull-Analyse durch
func runWeibullAnalysis(config Config, records []models.WindRecord) {
	fmt.Println("🌬️ Berechne Weibull-Verteilungen...")

	weibullRecords := interpolation.FilterRecordsFromYear(records, 2022)
	if len(weibullRecords) == 0 {
		weibullRecords = records
	}

	weibullInputs, _, err := weibull.BuildPlotsForLocation(config.LocationName, weibullRecords)
	if err != nil {
		fmt.Printf("⚠️ Weibull-Analyse für %s übersprungen: %v\n", config.LocationName, err)
		return
	}

	weibullPath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("weibull_analysis_%s.html", utils.SanitizeForFilename(config.LocationName)))
	if err := visualization.PlotWeibullDashboard(weibullInputs, weibullPath); err != nil {
		fmt.Printf("⚠️ Fehler beim Erstellen der Weibull-Analyse: %v\n", err)
		return
	}
	fmt.Printf("🌬️ Weibull-Analyse gespeichert: %s\n", weibullPath)
}