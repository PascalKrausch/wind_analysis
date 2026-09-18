package visualization

import (
	"fmt"
	"math"
	"sort"

	"wind_analysis/internal/analysis/interpolation"
)

// MetricRow repräsentiert eine Zeile in der Validierungstabelle.
type MetricRow struct {
	Location    string
	HeightM     float64
	MAE         float64
	RMSE        float64
	Correlation float64
	SampleCount int
}

// PlotValidationMetricsTable rendert eine saubere Ergebnistabelle für Validierungen.
func PlotValidationMetricsTable(rows []MetricRow, title, outputPath string) error {
	if len(rows) == 0 {
		return fmt.Errorf("keine Validierungsmetriken zum Rendern vorhanden")
	}

	// Sortieren nach Ort und Höhe
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Location == rows[j].Location {
			return rows[i].HeightM < rows[j].HeightM
		}
		return rows[i].Location < rows[j].Location
	})

	tableRows := make([][]string, len(rows))
	for i, r := range rows {
		tableRows[i] = []string{
			r.Location,
			fmt.Sprintf("%.0f", r.HeightM),
			fmt.Sprintf("%.4f", r.MAE),
			fmt.Sprintf("%.4f", r.RMSE),
			fmt.Sprintf("%.4f", r.Correlation),
			fmt.Sprintf("%d", r.SampleCount),
		}
	}

	return RenderTable(
		outputPath,
		title,
		"Modellvalidierung (MAE, RMSE, Korrelation)",
		[]string{"Ort", "Höhe [m]", "MAE", "RMSE", "Korrelation", "n"},
		tableRows,
	)
}

// PlotErrorDistributionShapeByHeight plottet Skewness und Kurtosis der Fehler je Höhe (für einen Ort).
func PlotErrorDistributionShapeByHeight(results map[string]map[float64]interpolation.ValidationResult, location, outputPath string) error {
	heightMap, ok := results[location]
	if !ok || len(heightMap) == 0 {
		return fmt.Errorf("keine Daten für Ort '%s' vorhanden", location)
	}

	heights := make([]float64, 0, len(heightMap))
	for h := range heightMap {
		heights = append(heights, h)
	}
	sort.Float64s(heights)

	xData := make([]string, 0, len(heights))
	skewness := make([]float64, 0, len(heights))
	kurtosis := make([]float64, 0, len(heights))

	for _, h := range heights {
		r := heightMap[h]
		xData = append(xData, fmt.Sprintf("%.0f", h))

		s := r.ErrorSummary.Skewness
		k := r.ErrorSummary.Kurtosis
		if math.IsNaN(s) {
			s = 0
		}
		if math.IsNaN(k) {
			k = 0
		}
		skewness = append(skewness, s)
		kurtosis = append(kurtosis, k)
	}

	cfg := DefaultChartConfig()
	cfg.Title = "Fehlerverteilung je Höhe"
	cfg.Subtitle = "Skewness & Kurtosis (" + location + ")"
	cfg.XAxisName = "Höhe [m]"
	cfg.YAxisName = "Wert"

	bar := CreateBarChart(cfg)
	AddBarSeries(bar, "Skewness", xData, skewness)
	AddBarSeries(bar, "Kurtosis", xData, kurtosis)

	return RenderToFile(bar, outputPath)
}
