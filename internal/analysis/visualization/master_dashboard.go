package visualization

import (
	"fmt"
	"math"
	"sort"
	"time"

	"wind_analysis/internal/analysis/interpolation"
	"wind_analysis/internal/analysis/statistics"
	"wind_analysis/models"
)

// MasterDashboardData enthält alle Analysedaten für mehrere Standorte
// für das Master-Dashboard.
type MasterDashboardData struct {
	// Metadaten
	Locations []string
	StartTime time.Time
	EndTime   time.Time

	// Windgeschwindigkeits-Daten (zeitliche Verläufe)
	WindSpeedData map[string][]TimeSeriesPoint // key: locationName

	// Hellmann-Exponenten (zeitliche Verläufe)
	HellmannData map[string][]TimeSeriesPoint // key: locationName

	// Verteilungs-Daten (für verschiedene Höhen und Modelle)
	DistributionData map[string][]DistributionPlotInput // key: locationName

	// Validierungs-Metriken
	ValidationData map[string][]MetricRow // key: locationName

	// Error-Shape-Daten (für verschiedene Höhen)
	ErrorShapeData map[string]map[float64]interpolation.ValidationResult // key: locationName
}

// NewMasterDashboardData erstellt eine neue leere MasterDashboardData Struktur
func NewMasterDashboardData() *MasterDashboardData {
	return &MasterDashboardData{
		Locations:        make([]string, 0),
		WindSpeedData:    make(map[string][]TimeSeriesPoint),
		HellmannData:     make(map[string][]TimeSeriesPoint),
		DistributionData: make(map[string][]DistributionPlotInput),
		ValidationData:   make(map[string][]MetricRow),
		ErrorShapeData:   make(map[string]map[float64]interpolation.ValidationResult),
	}
}

// AddLocation fügt einen neuen Standort zur Master-Dashboard-Struktur hinzu
func (m *MasterDashboardData) AddLocation(locationName string) {
	if !contains(m.Locations, locationName) {
		m.Locations = append(m.Locations, locationName)
		sort.Strings(m.Locations)
	}
}

// AddWindSpeedData fügt Windgeschwindigkeits-Daten für einen Standort hinzu
func (m *MasterDashboardData) AddWindSpeedData(locationName string, points []TimeSeriesPoint) {
	m.AddLocation(locationName)
	m.WindSpeedData[locationName] = append([]TimeSeriesPoint(nil), points...)
}

// AddHellmannData fügt Hellmann-Exponenten-Daten für einen Standort hinzu
func (m *MasterDashboardData) AddHellmannData(locationName string, points []TimeSeriesPoint) {
	m.AddLocation(locationName)
	m.HellmannData[locationName] = append([]TimeSeriesPoint(nil), points...)
}

// AddDistributionData fügt Verteilungs-Daten für einen Standort hinzu
func (m *MasterDashboardData) AddDistributionData(locationName string, inputs []DistributionPlotInput) {
	m.AddLocation(locationName)
	m.DistributionData[locationName] = cloneDistributionInputs(inputs)
}

// AddValidationData fügt Validierungs-Metriken für einen Standort hinzu
func (m *MasterDashboardData) AddValidationData(locationName string, rows []MetricRow) {
	m.AddLocation(locationName)
	m.ValidationData[locationName] = append([]MetricRow(nil), rows...)
}

// AddErrorShapeData fügt Error-Shape-Daten für einen Standort hinzu
func (m *MasterDashboardData) AddErrorShapeData(locationName string, heightMap map[float64]interpolation.ValidationResult) {
	m.AddLocation(locationName)
	m.ErrorShapeData[locationName] = cloneValidationResults(heightMap)
}

// SetTimeRange setzt den globalen Zeitbereich für das Dashboard
func (m *MasterDashboardData) SetTimeRange(start, end time.Time) {
	m.StartTime = start
	m.EndTime = end
}

// GetLocationsForChart gibt die sortierten Standortnamen zurück
func (m *MasterDashboardData) GetLocationsForChart() []string {
	return append([]string(nil), m.Locations...)
}

// HasWindSpeedData prüft ob Windgeschwindigkeits-Daten für einen Standort vorhanden sind
func (m *MasterDashboardData) HasWindSpeedData(locationName string) bool {
	_, ok := m.WindSpeedData[locationName]
	return ok
}

// HasHellmannData prüft ob Hellmann-Daten für einen Standort vorhanden sind
func (m *MasterDashboardData) HasHellmannData(locationName string) bool {
	_, ok := m.HellmannData[locationName]
	return ok
}

// HasDistributionData prüft ob Verteilungs-Daten für einen Standort vorhanden sind
func (m *MasterDashboardData) HasDistributionData(locationName string) bool {
	_, ok := m.DistributionData[locationName]
	return ok
}

// HasValidationData prüft ob Validierungs-Daten für einen Standort vorhanden sind
func (m *MasterDashboardData) HasValidationData(locationName string) bool {
	_, ok := m.ValidationData[locationName]
	return ok
}

// HasErrorShapeData prüft ob Error-Shape-Daten für einen Standort vorhanden sind
func (m *MasterDashboardData) HasErrorShapeData(locationName string) bool {
	_, ok := m.ErrorShapeData[locationName]
	return ok
}

// GetWindSpeedTimeSeries konvertiert Windgeschwindigkeits-Daten in LocationTimeSeries Format
func (m *MasterDashboardData) GetWindSpeedTimeSeries(locationName string) LocationTimeSeries {
	points, ok := m.WindSpeedData[locationName]
	if !ok {
		return LocationTimeSeries{LocationName: locationName, Points: []TimeSeriesPoint{}}
	}
	return LocationTimeSeries{LocationName: locationName, Points: append([]TimeSeriesPoint(nil), points...)}
}

// GetHellmannTimeSeries konvertiert Hellmann-Daten in LocationTimeSeries Format
func (m *MasterDashboardData) GetHellmannTimeSeries(locationName string) LocationTimeSeries {
	points, ok := m.HellmannData[locationName]
	if !ok {
		return LocationTimeSeries{LocationName: locationName, Points: []TimeSeriesPoint{}}
	}
	return LocationTimeSeries{LocationName: locationName, Points: append([]TimeSeriesPoint(nil), points...)}
}

// GetDistributionInputs gibt die Verteilungs-Inputs für einen Standort zurück
func (m *MasterDashboardData) GetDistributionInputs(locationName string) []DistributionPlotInput {
	inputs, ok := m.DistributionData[locationName]
	if !ok {
		return []DistributionPlotInput{}
	}
	return cloneDistributionInputs(inputs)
}

// GetValidationRows gibt die Validierungs-Metriken für einen Standort zurück
func (m *MasterDashboardData) GetValidationRows(locationName string) []MetricRow {
	rows, ok := m.ValidationData[locationName]
	if !ok {
		return []MetricRow{}
	}
	return append([]MetricRow(nil), rows...)
}

// GetErrorShapeData gibt die Error-Shape-Daten für einen Standort zurück
func (m *MasterDashboardData) GetErrorShapeData(locationName string) map[float64]interpolation.ValidationResult {
	data, ok := m.ErrorShapeData[locationName]
	if !ok {
		return map[float64]interpolation.ValidationResult{}
	}
	return cloneValidationResults(data)
}

// GetAllWindSpeedTimeSeries gibt alle Windgeschwindigkeits-Zeitreihen zurück
func (m *MasterDashboardData) GetAllWindSpeedTimeSeries() []LocationTimeSeries {
	result := make([]LocationTimeSeries, 0, len(m.Locations))
	for _, loc := range m.Locations {
		if m.HasWindSpeedData(loc) {
			result = append(result, m.GetWindSpeedTimeSeries(loc))
		}
	}
	return result
}

// GetAllHellmannTimeSeries gibt alle Hellmann-Zeitreihen zurück
func (m *MasterDashboardData) GetAllHellmannTimeSeries() []LocationTimeSeries {
	result := make([]LocationTimeSeries, 0, len(m.Locations))
	for _, loc := range m.Locations {
		if m.HasHellmannData(loc) {
			result = append(result, m.GetHellmannTimeSeries(loc))
		}
	}
	return result
}

// GetAllValidationRows gibt alle Validierungs-Metriken zurück
func (m *MasterDashboardData) GetAllValidationRows() []MetricRow {
	result := make([]MetricRow, 0)
	for _, loc := range m.Locations {
		if m.HasValidationData(loc) {
			result = append(result, m.ValidationData[loc]...)
		}
	}
	return result
}

// GetAllDistributionInputs gibt alle Verteilungs-Inputs zurück
func (m *MasterDashboardData) GetAllDistributionInputs() []DistributionPlotInput {
	result := make([]DistributionPlotInput, 0)
	for _, loc := range m.Locations {
		if m.HasDistributionData(loc) {
			result = append(result, cloneDistributionInputs(m.DistributionData[loc])...)
		}
	}
	return result
}

// Hilfsfunktion zum Prüfen ob ein String in einem Slice enthalten ist
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// --- Mapping-Funktionen zur Konvertierung von existierenden Datenstrukturen ---

// MapLocationToMasterDashboard konvertiert die Daten eines einzelnen Standorts
// aus dem validation processor Format in das Master-Dashboard Format
func MapLocationToMasterDashboard(
	location models.Location,
	records []models.WindRecord,
	exponents []interpolation.HellmannExponentResult,
	validationByLocation map[string]map[float64]interpolation.ValidationResult,
	distributionInputs []DistributionPlotInput,
) (*MasterDashboardData, error) {
	master := NewMasterDashboardData()

	// Standort hinzufügen
	master.AddLocation(location.Name)

	// Windgeschwindigkeits-Daten mappen
	if windTS, err := buildWindSpeedTimeSeriesForLocation(location.Name, records); err == nil {
		master.AddWindSpeedData(location.Name, windTS.Points)
	}

	// Hellmann-Exponenten mappen
	hellmannTS := mapExponentsToTimeSeries(location.Name, exponents)
	master.AddHellmannData(location.Name, hellmannTS.Points)

	// Validierungs-Daten mappen
	if validationByLocation != nil {
		if heightMap, ok := validationByLocation[location.Name]; ok {
			rows := mapValidationToMetricRows(location.Name, heightMap)
			master.AddValidationData(location.Name, rows)
			master.AddErrorShapeData(location.Name, heightMap)
		}
	}

	// Verteilungs-Daten mappen
	if len(distributionInputs) > 0 {
		master.AddDistributionData(location.Name, distributionInputs)
	}

	// Zeitbereich aus den Daten ableiten
	if start, end, ok := recordTimeRange(records); ok {
		master.SetTimeRange(start, end)
	}

	return master, nil
}

// MapMultipleLocationsToMasterDashboard konvertiert Daten mehrerer Standorte
// in das Master-Dashboard Format
func MapMultipleLocationsToMasterDashboard(
	locations []models.Location,
	recordsMap map[string][]models.WindRecord,
	exponentsMap map[string][]interpolation.HellmannExponentResult,
	validationByLocation map[string]map[float64]interpolation.ValidationResult,
	distributionInputsMap map[string][]DistributionPlotInput,
) (*MasterDashboardData, error) {
	master := NewMasterDashboardData()

	var allTimes []time.Time

	for _, location := range locations {
		master.AddLocation(location.Name)

		records, hasRecords := recordsMap[location.Name]
		exponents, hasExponents := exponentsMap[location.Name]
		distributionInputs, hasDistribution := distributionInputsMap[location.Name]

		if hasRecords {
			for _, record := range records {
				if !record.Time.IsZero() {
					allTimes = append(allTimes, record.Time)
				}
			}
		}

		// Windgeschwindigkeits-Daten
		if hasRecords && len(records) > 0 {
			if windTS, err := buildWindSpeedTimeSeriesForLocation(location.Name, records); err == nil {
				master.AddWindSpeedData(location.Name, windTS.Points)
			}
		}

		// Hellmann-Exponenten
		if hasExponents && len(exponents) > 0 {
			hellmannTS := mapExponentsToTimeSeries(location.Name, exponents)
			master.AddHellmannData(location.Name, hellmannTS.Points)
		}

		// Validierungs-Daten
		if validationByLocation != nil {
			if heightMap, ok := validationByLocation[location.Name]; ok {
				rows := mapValidationToMetricRows(location.Name, heightMap)
				master.AddValidationData(location.Name, rows)
				master.AddErrorShapeData(location.Name, heightMap)
			}
		}

		// Verteilungs-Daten
		if hasDistribution && len(distributionInputs) > 0 {
			master.AddDistributionData(location.Name, distributionInputs)
		}
	}

	// Globalen Zeitbereich setzen
	if len(allTimes) > 0 {
		sort.Slice(allTimes, func(i, j int) bool { return allTimes[i].Before(allTimes[j]) })
		master.SetTimeRange(allTimes[0], allTimes[len(allTimes)-1])
	}

	return master, nil
}

// buildWindSpeedTimeSeriesForLocation extrahiert Windgeschwindigkeits-Werte aus Records
// und konvertiert sie in TimeSeriesPoints
func buildWindSpeedTimeSeriesForLocation(locName string, records []models.WindRecord) (LocationTimeSeries, error) {
	points := make([]TimeSeriesPoint, 0, len(records))

	for _, r := range records {
		t, v, ok := extractRecordTimeAndSpeed(r)
		if !ok || t.IsZero() || math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		points = append(points, TimeSeriesPoint{
			Time:  t,
			Value: v,
		})
	}

	if len(points) == 0 {
		return LocationTimeSeries{}, fmt.Errorf("keine Windgeschwindigkeitswerte aus geladenen Datensätzen extrahierbar")
	}

	sort.Slice(points, func(i, j int) bool { return points[i].Time.Before(points[j].Time) })

	return LocationTimeSeries{
		LocationName: locName,
		Points:       points,
	}, nil
}

// extractRecordTimeAndSpeed extrahiert Zeit und Windgeschwindigkeit aus einem WindRecord
func extractRecordTimeAndSpeed(record models.WindRecord) (time.Time, float64, bool) {
	t := record.Time

	// Versuche verschiedene Windgeschwindigkeits-Felder zu finden
	// Priorität: 10m, dann 80m, dann 100m
	v := 0.0
	ok := false

	for _, candidate := range []*float64{
		record.WindData.WindSpeed_10m,
		record.WindData.WindSpeed_80m,
		record.WindData.WindSpeed_100m,
	} {
		if candidate != nil && !math.IsNaN(*candidate) && !math.IsInf(*candidate, 0) && *candidate >= 0 {
			v = *candidate
			ok = true
			break
		}
	}

	return t, v, ok
}

// mapExponentsToTimeSeries konvertiert Hellmann-Exponenten in TimeSeriesPoints
// (diese Funktion existiert bereits im processor.go, wird hier für Konsistenz neu definiert)
func mapExponentsToTimeSeries(locName string, exponents []interpolation.HellmannExponentResult) LocationTimeSeries {
	sorted := append([]interpolation.HellmannExponentResult(nil), exponents...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Time.Before(sorted[j].Time) })

	pts := make([]TimeSeriesPoint, 0, len(sorted))
	for _, exp := range sorted {
		if exp.Time.IsZero() || math.IsNaN(exp.Alpha) || math.IsInf(exp.Alpha, 0) {
			continue
		}
		pts = append(pts, TimeSeriesPoint{
			Time:  exp.Time,
			Value: exp.Alpha,
		})
	}
	return LocationTimeSeries{
		LocationName: locName,
		Points:       pts,
	}
}

// mapValidationToMetricRows konvertiert Validierungs-Daten in MetricRows
// (diese Funktion existiert bereits im processor.go, wird hier für Konsistenz neu definiert)
func mapValidationToMetricRows(location string, heightMap map[float64]interpolation.ValidationResult) []MetricRow {
	heights := make([]float64, 0, len(heightMap))
	for height := range heightMap {
		heights = append(heights, height)
	}
	sort.Float64s(heights)

	rows := make([]MetricRow, 0, len(heights))
	for _, h := range heights {
		res := heightMap[h]
		rows = append(rows, MetricRow{
			Location:    location,
			HeightM:     h,
			MAE:         res.MAE,
			RMSE:        res.RMSE,
			Correlation: res.Correlation,
			SampleCount: res.SampleCount,
		})
	}
	return rows
}

func recordTimeRange(records []models.WindRecord) (time.Time, time.Time, bool) {
	var start, end time.Time
	for _, record := range records {
		if record.Time.IsZero() {
			continue
		}
		if start.IsZero() || record.Time.Before(start) {
			start = record.Time
		}
		if end.IsZero() || record.Time.After(end) {
			end = record.Time
		}
	}
	return start, end, !start.IsZero()
}

func cloneDistributionInputs(inputs []DistributionPlotInput) []DistributionPlotInput {
	cloned := append([]DistributionPlotInput(nil), inputs...)
	for i := range cloned {
		cloned[i].Histogram = append([]statistics.Bin(nil), inputs[i].Histogram...)
		cloned[i].EmpiricalCDF = append([]statistics.CDFPoint(nil), inputs[i].EmpiricalCDF...)
		cloned[i].FittedPDF = append([]statistics.DensityPoint(nil), inputs[i].FittedPDF...)
		cloned[i].FittedCDF = append([]statistics.CDFPoint(nil), inputs[i].FittedCDF...)
	}
	return cloned
}

func cloneValidationResults(results map[float64]interpolation.ValidationResult) map[float64]interpolation.ValidationResult {
	cloned := make(map[float64]interpolation.ValidationResult, len(results))
	for height, result := range results {
		result.PredictedSummary = cloneDescriptiveStats(result.PredictedSummary)
		result.ActualSummary = cloneDescriptiveStats(result.ActualSummary)
		result.ErrorSummary = cloneDescriptiveStats(result.ErrorSummary)
		cloned[height] = result
	}
	return cloned
}

func cloneDescriptiveStats(stats interpolation.DescriptiveStats) interpolation.DescriptiveStats {
	if stats.Quantiles != nil {
		quantiles := make(map[float64]float64, len(stats.Quantiles))
		for key, value := range stats.Quantiles {
			quantiles[key] = value
		}
		stats.Quantiles = quantiles
	}
	if stats.Percentil != nil {
		percentiles := make(map[float64]float64, len(stats.Percentil))
		for key, value := range stats.Percentil {
			percentiles[key] = value
		}
		stats.Percentil = percentiles
	}
	return stats
}
