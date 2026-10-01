package validation

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/fitting"
	"wind_analysis/internal/analysis/interpolation"
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

	// 2a. Windgeschwindigkeit-Timeline (optional)
	if windTS, err := buildWindSpeedTimeSeriesForLocation(config.LocationName, records); err != nil {
		fmt.Printf("⚠️ Windgeschwindigkeit-Timeline übersprungen (%s): %v\n", config.LocationName, err)
	} else {
		windPath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("windspeed_timeline_%s.html", utils.SanitizeForFilename(config.LocationName)))
		if err := visualization.PlotMultiLocationTimeline(
			[]visualization.LocationTimeSeries{windTS},
			"Windgeschwindigkeit über Zeit",
			"Windgeschwindigkeit (m/s)",
			windPath,
		); err != nil {
			if optErr := handleOptionalStep(config, "Windgeschwindigkeit-Timeline fehlgeschlagen", err); optErr != nil {
				return optErr
			}
		} else {
			fmt.Printf("📈 Windgeschwindigkeit-Timeline gespeichert: %s\n", windPath)
		}
	}

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
	recordsByLocation := make(map[string][]models.WindRecord, len(locationList))
	exponentsByLocation := make(map[string][]interpolation.HellmannExponentResult, len(locationList))
	var timeSeriesList []visualization.LocationTimeSeries
	var windSpeedSeriesList []visualization.LocationTimeSeries

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
		recordsByLocation[location.Name] = records

		if windTS, err := buildWindSpeedTimeSeriesForLocation(location.Name, records); err == nil {
			windSpeedSeriesList = append(windSpeedSeriesList, windTS)
		} else {
			fmt.Printf("  ⚠️ Windgeschwindigkeit für %s nicht geplottet: %v\n", location.Name, err)
		}

		exponents := interpolation.CalculateHellmannExponentsForDataset(records)
		if len(exponents) == 0 {
			fmt.Printf("  ⚠️ Keine gültigen Exponenten für %s\n", location.Name)
			continue
		}
		exponentsByLocation[location.Name] = exponents
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

	vizInputs := make([]visualization.DistributionPlotInput, 0, len(allPlotInputs))
	distributionInputsByLocation := make(map[string][]visualization.DistributionPlotInput)
	for _, p := range allPlotInputs {
		input := mapFittingToVizInput(p)
		vizInputs = append(vizInputs, input)
		distributionInputsByLocation[input.LocationName] = append(distributionInputsByLocation[input.LocationName], input)
	}

	validationByLocation := interpolation.ValidatePowerLawModelByLocation(allRecordsFrom2022, allExponents)
	master, err := visualization.MapMultipleLocationsToMasterDashboard(
		locationList,
		recordsByLocation,
		exponentsByLocation,
		validationByLocation,
		distributionInputsByLocation,
	)
	if err != nil {
		return fmt.Errorf("Master-Dashboard-Daten konnten nicht zusammengestellt werden: %w", err)
	}
	masterDashboardPath := utils.BuildOutputPath(config.OutputDir, "master_dashboard.html")
	if err := visualization.PlotMasterDashboard(master, masterDashboardPath); err != nil {
		return fmt.Errorf("Master-Dashboard konnte nicht erstellt werden: %w", err)
	}
	fmt.Printf("📊 Master-Dashboard gespeichert: %s\n", masterDashboardPath)

	// Standortvergleich der Exponenten zeichnen
	if len(timeSeriesList) > 0 {
		comparisonPath := utils.BuildOutputPath(config.OutputDir, "location_comparison.html")
		if err := visualization.PlotMultiLocationTimeline(timeSeriesList, "Hellmann-Exponent: Standortvergleich", "Hellmann-Exponent α", comparisonPath); err != nil {
			return fmt.Errorf("fehler beim Erstellen des Standortvergleichs: %w", err)
		}
		fmt.Printf("📈 Standortvergleich gespeichert: %s\n", comparisonPath)
	} else {
		fmt.Println("ℹ️ Hellmann-Standortvergleich übersprungen (keine gültigen Exponenten vorhanden).")
	}

	// Standortvergleich der Windgeschwindigkeit zeichnen
	if len(windSpeedSeriesList) >= 2 {
		windComparisonPath := utils.BuildOutputPath(config.OutputDir, "windspeed_location_comparison.html")
		if err := visualization.PlotMultiLocationTimeline(
			windSpeedSeriesList,
			"Windgeschwindigkeit: Standortvergleich",
			"Windgeschwindigkeit (m/s)",
			windComparisonPath,
		); err != nil {
			if optErr := handleOptionalStep(config, "Windgeschwindigkeit-Standortvergleich fehlgeschlagen", err); optErr != nil {
				return optErr
			}
		} else {
			fmt.Printf("💨 Windgeschwindigkeit-Vergleich gespeichert: %s\n", windComparisonPath)
		}
	} else {
		fmt.Println("ℹ️ Windgeschwindigkeit-Vergleich übersprungen (weniger als 2 Standorte mit gültigen Zeitreihen).")
	}

	// Globale Validierungsmetriken
	if len(allRecordsFrom2022) > 0 && len(allExponents) > 0 {
		if err := runGlobalValidationMetrics(config, allRecordsFrom2022, allExponents); err != nil {
			return handleOptionalStep(config, "globale Validierung fehlgeschlagen", err)
		}
	} else {
		fmt.Println("ℹ️ Globale Validierung übersprungen (keine geeigneten Daten vorhanden).")
	}

	// Verteilungs-Übersicht erstellen
	if len(vizInputs) > 0 {
		distributionComparisonPath := utils.BuildOutputPath(config.OutputDir, "distribution_location_comparison.html")
		if err := visualization.PlotDistributionDashboard(vizInputs, distributionComparisonPath); err != nil {
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
	heights := extractHeightsForLocation(validationByLocation, config.LocationName)
	if len(heights) > 0 {
		errorShapePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_error_shape_%s.html", utils.SanitizeForFilename(config.LocationName)))
		if err := visualization.PlotErrorDistributionShapeByHeight(validationByLocation, config.LocationName, errorShapePath); err != nil {
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

		heights := extractHeightsForLocation(validationByLocation, loc)
		if len(heights) > 0 {
			errorShapePath := utils.BuildOutputPath(config.OutputDir, fmt.Sprintf("validation_error_shape_%s.html", utils.SanitizeForFilename(loc)))
			if err := visualization.PlotErrorDistributionShapeByHeight(validationByLocation, loc, errorShapePath); err != nil {
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
	sorted := append([]interpolation.HellmannExponentResult(nil), exponents...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Time.Before(sorted[j].Time) })

	pts := make([]visualization.TimeSeriesPoint, 0, len(sorted))
	for _, exp := range sorted {
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

func extractHeightsForLocation(validation map[string]map[float64]interpolation.ValidationResult, loc string) []float64 {
	heightMap, ok := validation[loc]
	if !ok || len(heightMap) == 0 {
		return nil
	}

	heights := make([]float64, 0, len(heightMap))
	for h := range heightMap {
		heights = append(heights, h)
	}
	sort.Float64s(heights)

	return heights
}

func mapFittingToVizInput(p fitting.AnalysisPlotInput) visualization.DistributionPlotInput {
	return visualization.DistributionPlotInput{
		SeriesName:   p.SeriesName,
		LocationName: p.LocationName,
		HeightM:      p.HeightM,
		Distribution: p.Model,
		ModelName:    p.FitterName,
		Metrics:      p.Metrics,
		Histogram:    p.Histogram,
		EmpiricalCDF: p.EmpiricalCDF,
		FittedPDF:    p.FittedPDF,
		FittedCDF:    p.FittedCDF,
	}
}

func buildWindSpeedTimeSeriesForLocation(locName string, records []models.WindRecord) (visualization.LocationTimeSeries, error) {
	points := make([]visualization.TimeSeriesPoint, 0, len(records))

	for _, r := range records {
		t, v, ok := extractRecordTimeAndSpeed(r)
		if !ok || t.IsZero() || math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		points = append(points, visualization.TimeSeriesPoint{
			Time:  t,
			Value: v,
		})
	}

	if len(points) == 0 {
		return visualization.LocationTimeSeries{}, fmt.Errorf("keine Windgeschwindigkeitswerte aus geladenen Datensätzen extrahierbar")
	}

	sort.Slice(points, func(i, j int) bool { return points[i].Time.Before(points[j].Time) })

	return visualization.LocationTimeSeries{
		LocationName: locName,
		Points:       points,
	}, nil
}

func extractRecordTimeAndSpeed(record models.WindRecord) (time.Time, float64, bool) {
	rv := reflect.ValueOf(record)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return time.Time{}, 0, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return time.Time{}, 0, false
	}

	t, okT := extractTimeFromStruct(rv, []string{
		"Time", "Timestamp", "DateTime", "Datetime", "MeasurementTime", "MeasuredAt",
	})
	v, okV := extractFloatFromStruct(rv, []string{
		"WindSpeed", "WindSpeedMS", "WindSpeedMs", "WS", "Speed", "Velocity", "V",
	})

	return t, v, okT && okV
}

func extractTimeFromStruct(rv reflect.Value, names []string) (time.Time, bool) {
	// 1) bevorzugte Feldnamen (case-insensitive)
	for _, n := range names {
		if f, ok := fieldByNameFold(rv, n); ok {
			if t, ok := valueToTime(f); ok {
				return t, true
			}
		}
	}

	// 2) Fallback: erstes plausibles Zeitfeld
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		if !looksLikeTimeName(sf.Name) {
			continue
		}
		if t, ok := valueToTime(rv.Field(i)); ok {
			return t, true
		}
	}

	return time.Time{}, false
}

func extractFloatFromStruct(rv reflect.Value, names []string) (float64, bool) {
	// 1) bevorzugte Feldnamen (case-insensitive)
	for _, n := range names {
		if f, ok := fieldByNameFold(rv, n); ok {
			if v, ok := valueToFloat(f); ok && isPlausibleWindSpeed(v) {
				return v, true
			}
		}
	}

	// 2) Fallback: rekursiv plausible Windspeed-Felder sammeln
	candidates := make([]float64, 0, 8)
	collectLikelyWindSpeeds(rv, "", 0, &candidates)
	if len(candidates) == 0 {
		return 0, false
	}

	sum := 0.0
	for _, v := range candidates {
		sum += v
	}
	return sum / float64(len(candidates)), true
}

func fieldByNameFold(rv reflect.Value, name string) (reflect.Value, bool) {
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if strings.EqualFold(sf.Name, name) {
			return rv.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func valueToTime(v reflect.Value) (time.Time, bool) {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return time.Time{}, false
		}
		v = v.Elem()
	}

	if v.Type() == reflect.TypeOf(time.Time{}) {
		return v.Interface().(time.Time), true
	}
	if v.Kind() == reflect.String {
		s := strings.TrimSpace(v.String())
		for _, layout := range []string{
			time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02",
		} {
			if t, err := time.Parse(layout, s); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func valueToFloat(v reflect.Value) (float64, bool) {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0, false
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Convert(reflect.TypeOf(float64(0))).Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.String:
		f, err := strconv.ParseFloat(strings.TrimSpace(v.String()), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func collectLikelyWindSpeeds(v reflect.Value, fieldName string, depth int, out *[]float64) {
	if depth > 4 || !v.IsValid() {
		return
	}

	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		rt := v.Type()
		for i := 0; i < v.NumField(); i++ {
			sf := rt.Field(i)
			if sf.PkgPath != "" {
				continue
			}
			collectLikelyWindSpeeds(v.Field(i), sf.Name, depth+1, out)
		}
	case reflect.Map:
		if !looksLikeWindName(fieldName) {
			return
		}
		it := v.MapRange()
		for it.Next() {
			collectLikelyWindSpeeds(it.Value(), fieldName, depth+1, out)
		}
	case reflect.Slice, reflect.Array:
		if !looksLikeWindName(fieldName) {
			return
		}
		for i := 0; i < v.Len(); i++ {
			collectLikelyWindSpeeds(v.Index(i), fieldName, depth+1, out)
		}
	default:
		if !looksLikeWindName(fieldName) {
			return
		}
		if f, ok := valueToFloat(v); ok && isPlausibleWindSpeed(f) {
			*out = append(*out, f)
		}
	}
}

func looksLikeTimeName(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "time") || strings.Contains(n, "date") || strings.Contains(n, "stamp")
}

func looksLikeWindName(name string) bool {
	n := strings.ToLower(name)
	if n == "v" || strings.HasPrefix(n, "ws") {
		return true
	}
	if strings.Contains(n, "dir") || strings.Contains(n, "direction") {
		return false
	}
	return strings.Contains(n, "wind") || strings.Contains(n, "speed") || strings.Contains(n, "velo")
}

func isPlausibleWindSpeed(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 150
}
