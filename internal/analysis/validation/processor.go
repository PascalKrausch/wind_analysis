package validation

import (
	"context"
	"fmt"
	"sort"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/fitting"
	"wind_analysis/internal/analysis/interpolation"
	"wind_analysis/internal/analysis/statistics"
	"wind_analysis/internal/analysis/visualization"

	"wind_analysis/internal/database"
	"wind_analysis/internal/utils"
	"wind_analysis/models"
)

// Config enthält die Konfiguration für die Validierungs- und Verteilungsanalyse.
type Config struct {
	StartTime    string                // Startzeit der Analyse
	EndTime      string                // Endzeit der Analyse
	OutputDir    string                // Verzeichnis für Ausgabedateien
	LocationName string                // Optional für einzelne Standort-Analyse
	Fitters      []distribution.Fitter // Standard-Verteilungsfitter (z. B. Weibull, LogNormal, Gamma)
	Strict       bool                  // true: bei optionalen Analyse-/Plot-Fehlern sofort abbrechen
}

// DefaultFitters liefert die Standard-Verteilungsmodelle für Windanalysen.
func DefaultFitters() []distribution.Fitter {
	return []distribution.Fitter{
		distribution.WeibullFitter{},
		distribution.LogNormalFitter{},
		distribution.GammaFitter{},
	}
}

// validateConfig überprüft die Konfiguration auf gültige Werte.
func validateConfig(config Config) error {
	if config.StartTime == "" || config.EndTime == "" {
		return fmt.Errorf("StartTime und EndTime müssen gesetzt sein")
	}
	if config.OutputDir == "" {
		return fmt.Errorf("OutputDir muss gesetzt sein")
	}
	return nil
}

// handleOptionalStep behandelt optionale Schritte mit Fehlerbehandlung.
func handleOptionalStep(config Config, step string, err error) error {
	if err == nil {
		return nil
	}
	wrapped := fmt.Errorf("%s: %w", step, err)
	if config.Strict {
		return wrapped
	}
	fmt.Printf("⚠️ %v\n", wrapped)
	return nil
}

// RunSingleLocationAnalysis führt die Analyse für einen einzelnen Standort durch.
func RunSingleLocationAnalysis(ctx context.Context, db *database.DB, config Config) error {
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("ungültige Konfiguration: %w", err)
	}
	if config.LocationName == "" {
		return fmt.Errorf("LocationName muss für Einzelstandort-Analyse gesetzt sein")
	}

	fmt.Printf("📊 Lade Winddaten für %s von %s bis %s...\n", config.LocationName, config.StartTime, config.EndTime)

	// 1. Winddaten laden
	records, err := db.LoadWindData(ctx, config.LocationName, config.StartTime, config.EndTime)
	if err != nil {
		return fmt.Errorf("fehler beim Laden der Winddaten: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("keine Daten für den angegebenen Zeitraum gefunden")
	}

	fmt.Printf("✅ %d Datensätze geladen\n", len(records))

	// 2. Hellmann-Exponenten berechnen
	fmt.Println("🔬 Berechne Hellmann-Exponenten...")
	exponents := interpolation.CalculateHellmannExponentsForDataset(records)
	if len(exponents) == 0 {
		return fmt.Errorf("keine gültigen Hellmann-Exponenten berechnet (Standort: %s)", config.LocationName)
	}
	fmt.Printf("✅ %d gültige Hellmann-Exponenten berechnet\n", len(exponents))

	// 3. Timeline-Visualisierung erstellen (mittels generischem LocationTimeSeries)
	timelinePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("hellmann_timeline_%s.html", utils.SanitizeForFilename(config.LocationName)))

	timeSeries := mapExponentsToTimeSeries(config.LocationName, exponents)
	if err := visualization.PlotMultiLocationTimeline([]visualization.LocationTimeSeries{timeSeries}, "Hellmann-Exponent über Zeit", "Hellmann-Exponent α", timelinePath); err != nil {
		return fmt.Errorf("fehler beim Erstellen der Timeline: %w", err)
	}
	fmt.Printf("📈 Timeline gespeichert: %s\n", timelinePath)

	// 4. Validierungsmetriken für Power-Law Modell
	if err := runValidationMetrics(config, records, exponents); err != nil {
		return handleOptionalStep(config, "Validierungsmetriken fehlgeschlagen", err)
	}

	// 5. Generische Verteilungsanalyse (Weibull, LogNormal etc.)
	if err := runDistributionAnalysis(config, records); err != nil {
		return handleOptionalStep(config, "Verteilungsanalyse fehlgeschlagen", err)
	}

	return nil
}

// RunLocationComparison führt den Vergleich zwischen allen Standorten durch.
func RunLocationComparison(ctx context.Context, db *database.DB, config Config, locationList []models.Location) error {
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("ungültige Konfiguration: %w", err)
	}
	if len(locationList) == 0 {
		return fmt.Errorf("locationList ist leer")
	}

	var allExponents []interpolation.HellmannExponentResult
	var allRecordsFrom2022 []models.WindRecord
	var allPlotInputs []fitting.AnalysisPlotInput
	var timeSeriesList []visualization.LocationTimeSeries

	fitters := config.Fitters
	if len(fitters) == 0 {
		fitters = DefaultFitters()
	}

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
		if len(exponents) == 0 {
			fmt.Printf("  ⚠️ Keine gültigen Exponenten für %s\n", location.Name)
			continue
		}
		allExponents = append(allExponents, exponents...)
		timeSeriesList = append(timeSeriesList, mapExponentsToTimeSeries(location.Name, exponents))

		records2022 := interpolation.FilterRecordsFromYear(records, 2022)
		if len(records2022) == 0 {
			records2022 = records
		}
		allRecordsFrom2022 = append(allRecordsFrom2022, records2022...)

		// Best-Fit-Analyse über alle Höhen
		inputs, err := fitting.FindBestFitsForLocation(location.Name, records2022, fitters, nil)
		if err != nil {
			fmt.Printf("  ⚠️ Best-Fit-Analyse fehlgeschlagen für %s: %v\n", location.Name, err)
		} else {
			allPlotInputs = append(allPlotInputs, inputs...)
		}

		fmt.Printf("  ✅ %d Exponenten für %s\n", len(exponents), location.Name)
	}

	if len(allExponents) == 0 {
		return fmt.Errorf("keine Exponenten für Standortvergleich gesammelt")
	}

	// Standortvergleich der Exponenten zeichnen
	comparisonPath := utils.BuildOutputPath(config.OutputDir, "location_comparison.html")
	if err := visualization.PlotMultiLocationTimeline(timeSeriesList, "Hellmann-Exponent: Standortvergleich", "Hellmann-Exponent α", comparisonPath); err != nil {
		return fmt.Errorf("fehler beim Erstellen des Standortvergleichs: %w", err)
	}
	fmt.Printf("📈 Standortvergleich gespeichert: %s\n", comparisonPath)

	// Globale Validierungsmetriken
	if err := runGlobalValidationMetrics(config, allRecordsFrom2022, allExponents); err != nil {
		return handleOptionalStep(config, "globale Validierung fehlgeschlagen", err)
	}

	// Verteilungs-Übersicht erstellen
	if len(allPlotInputs) > 0 {
		vizInputs := make([]visualization.DistributionPlotInput, 0, len(allPlotInputs))
		for _, p := range allPlotInputs {
			vizInputs = append(vizInputs, mapFittingToVizInput(p))
		}

		distributionComparisonPath := utils.BuildOutputPath(config.OutputDir, "distribution_location_comparison.html")
		if err := visualization.PlotDistributionComparisonOverview(vizInputs, distributionComparisonPath); err != nil {
			fmt.Printf("⚠️ Fehler beim Erstellen des Verteilungsvergleichs: %v\n", err)
		} else {
			fmt.Printf("🌬️ Verteilungsvergleich gespeichert: %s\n", distributionComparisonPath)
		}
	}

	return nil
}

// runValidationMetrics führt die Validierungsmetrik-Berechnung durch.
func runValidationMetrics(config Config, records []models.WindRecord, exponents []interpolation.HellmannExponentResult) error {
	fmt.Println("🔍 Validiere Power-Law Modell...")

	recordsFrom2022 := interpolation.FilterRecordsFromYear(records, 2022)
	if len(recordsFrom2022) == 0 {
		return fmt.Errorf("keine Daten ab 2022 für Validierung verfügbar")
	}

	validationByLocation := interpolation.ValidatePowerLawModelByLocation(recordsFrom2022, exponents)
	if len(validationByLocation) == 0 {
		return fmt.Errorf("keine Validierungsergebnisse verfügbar")
	}

	rows := mapValidationToMetricRows(validationByLocation)

	metricsPath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_metrics_by_location_%s.html", utils.SanitizeForFilename(config.LocationName)))
	if err := visualization.PlotValidationMetricsTable(rows, "Validierungsmetriken (Power-Law)", metricsPath); err != nil {
		return fmt.Errorf("Metrik-Tabelle: %w", err)
	}
	fmt.Printf("📋 Validierungsmetriken (Tabelle) gespeichert: %s\n", metricsPath)

	// Error-Shape-Chart zeichnen
	heights, errorStats := extractErrorStatsForLocation(validationByLocation, config.LocationName)
	if len(heights) > 0 {
		errorShapePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_error_shape_%s.html", utils.SanitizeForFilename(config.LocationName)))
		if err := visualization.PlotErrorDistributionShape(config.LocationName, heights, errorStats, errorShapePath); err != nil {
			return fmt.Errorf("Error-Shape-Chart (%s): %w", config.LocationName, err)
		}
		fmt.Printf("📊 Error-Shape-Chart gespeichert: %s\n", errorShapePath)
	}

	return nil
}

// runGlobalValidationMetrics führt die globale Validierung durch.
func runGlobalValidationMetrics(config Config, allRecordsFrom2022 []models.WindRecord, allExponents []interpolation.HellmannExponentResult) error {
	if len(allRecordsFrom2022) == 0 {
		return fmt.Errorf("keine Datensätze für globale Validierung")
	}

	validationByLocation := interpolation.ValidatePowerLawModelByLocation(allRecordsFrom2022, allExponents)
	if len(validationByLocation) == 0 {
		return fmt.Errorf("keine globalen Validierungsergebnisse")
	}

	rows := mapValidationToMetricRows(validationByLocation)

	metricsPath := utils.BuildOutputPath(config.OutputDir, "validation_metrics_by_height_and_location.html")
	if err := visualization.PlotValidationMetricsTable(rows, "Globale Validierungsmetriken (Power-Law)", metricsPath); err != nil {
		return fmt.Errorf("globale Metrik-Tabelle: %w", err)
	}
	fmt.Printf("📋 Globale Validierungsmetriken gespeichert: %s\n", metricsPath)

	locations := make([]string, 0, len(validationByLocation))
	for loc := range validationByLocation {
		locations = append(locations, loc)
	}
	sort.Strings(locations)

	for _, loc := range locations {
		fmt.Printf("  ✅ Metriken vorhanden für: %s\n", loc)

		heights, errorStats := extractErrorStatsForLocation(validationByLocation, loc)
		if len(heights) > 0 {
			errorShapePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_error_shape_%s.html", utils.SanitizeForFilename(loc)))
			if err := visualization.PlotErrorDistributionShape(loc, heights, errorStats, errorShapePath); err != nil {
				return fmt.Errorf("globales Error-Shape-Chart (%s): %w", loc, err)
			}
			fmt.Printf("  📊 Error-Shape-Chart gespeichert: %s\n", errorShapePath)
		}
	}

	return nil
}

// runDistributionAnalysis führt die Verteilungsanalyse universell durch.
func runDistributionAnalysis(config Config, records []models.WindRecord) error {
	fmt.Println("🌬️ Berechne Verteilungs-Fits...")

	fitRecords := interpolation.FilterRecordsFromYear(records, 2022)
	if len(fitRecords) == 0 {
		fitRecords = records
	}

	fitters := config.Fitters
	if len(fitters) == 0 {
		fitters = DefaultFitters()
	}

	plotInputs, err := fitting.FindBestFitsForLocation(config.LocationName, fitRecords, fitters, nil)
	if err != nil {
		return fmt.Errorf("Best-Fit-Ermittlung (%s): %w", config.LocationName, err)
	}

	vizInputs := make([]visualization.DistributionPlotInput, 0, len(plotInputs))
	for _, p := range plotInputs {
		vizInputs = append(vizInputs, mapFittingToVizInput(p))
	}

	dashboardPath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("distribution_analysis_%s.html", utils.SanitizeForFilename(config.LocationName)))
	if err := visualization.PlotDistributionDashboard(vizInputs, dashboardPath); err != nil {
		return fmt.Errorf("Verteilungs-Dashboard: %w", err)
	}
	fmt.Printf("🌬️ Verteilungs-Dashboard gespeichert: %s\n", dashboardPath)

	return nil
}

// --- Hilfsfunktionen für das Mapping von Analyse-Daten auf Visualisierungs-Daten ---

func mapExponentsToTimeSeries(locName string, exponents []interpolation.HellmannExponentResult) visualization.LocationTimeSeries {
	pts := make([]visualization.TimeSeriesPoint, 0, len(exponents))
	for _, exp := range exponents {
		pts = append(pts, visualization.TimeSeriesPoint{
			Time:  exp.Time,
			Value: exp.Alpha,
		})
	}
	return visualization.LocationTimeSeries{
		LocationName: locName,
		Points:       pts,
	}
}

func mapValidationToMetricRows(validation map[string]map[float64]interpolation.ValidationResult) []visualization.MetricRow {
	var rows []visualization.MetricRow
	for loc, heightMap := range validation {
		for h, res := range heightMap {
			rows = append(rows, visualization.MetricRow{
				Location:    loc,
				HeightM:     h,
				MAE:         res.MAE,
				RMSE:        res.RMSE,
				Correlation: res.Correlation,
				SampleCount: res.SampleCount,
			})
		}
	}
	return rows
}

func extractErrorStatsForLocation(validation map[string]map[float64]interpolation.ValidationResult, loc string) ([]float64, []statistics.Summary) {
	heightMap, ok := validation[loc]
	if !ok || len(heightMap) == 0 {
		return nil, nil
	}

	heights := make([]float64, 0, len(heightMap))
	for h := range heightMap {
		heights = append(heights, h)
	}
	sort.Float64s(heights)

	stats := make([]statistics.Summary, len(heights))
	for i, h := range heights {
		stats[i] = heightMap[h].ErrorSummary
	}

	return heights, stats
}

func mapFittingToVizInput(p fitting.AnalysisPlotInput) visualization.DistributionPlotInput {
	return visualization.DistributionPlotInput{
		SeriesName:   p.SeriesName,
		LocationName: p.LocationName,
		HeightM:      p.HeightM,
		Distribution: p.FitterName,
		ModelName:    p.Model,
		Metrics:      p.Metrics,
		Histogram:    p.Histogram,
		EmpiricalCDF: p.EmpiricalCDF,
		FittedPDF:    p.FittedPDF,
		FittedCDF:    p.FittedCDF,
	}
}
