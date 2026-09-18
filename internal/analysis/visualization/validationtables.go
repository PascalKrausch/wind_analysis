package visualization

import (
	"fmt"
	"sort"
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
