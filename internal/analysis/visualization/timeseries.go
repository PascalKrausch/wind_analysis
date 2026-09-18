package visualization

import (
	"fmt"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

// TimeSeriesPoint repräsentiert einen einzelnen Datenpunkt über die Zeit.
type TimeSeriesPoint struct {
	Time  time.Time
	Value float64
}

// LocationTimeSeries bündelt eine Zeitreihe für einen bestimmten Standort.
type LocationTimeSeries struct {
	LocationName string
	Points       []TimeSeriesPoint
}

// PlotMultiLocationTimeline zeichnet den zeitlichen Verlauf beliebiger Kennzahlen für mehrere Standorte.
func PlotMultiLocationTimeline(seriesList []LocationTimeSeries, title, yAxisName, outputPath string) error {
	if len(seriesList) == 0 {
		return fmt.Errorf("keine Zeitreihendaten für Standortvergleich vorhanden")
	}

	// 1. Alle eindeutigen Zeitstempel sammeln und sortieren
	allTimesMap := make(map[time.Time]struct{})
	for _, series := range seriesList {
		for _, pt := range series.Points {
			allTimesMap[pt.Time] = struct{}{}
		}
	}

	times := make([]time.Time, 0, len(allTimesMap))
	for t := range allTimesMap {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	xData := ConvertTimesToXAxisHourly(times)

	// 2. Chart initialisieren
	cfg := DefaultChartConfig()
	cfg.Title = title
	cfg.XAxisName = "Zeit"
	cfg.YAxisName = yAxisName
	cfg.EnableDataZoom = true

	line := CreateLineChart(cfg)
	line.SetXAxis(xData)

	// 3. Serien hinzufügen
	for _, series := range seriesList {
		// Value-Lookup-Map für schnelle Zuordnung zur Zeitachse
		valMap := make(map[time.Time]float64, len(series.Points))
		for _, pt := range series.Points {
			valMap[pt.Time] = pt.Value
		}

		ySeries := make([]opts.LineData, len(times))
		for i, t := range times {
			if val, ok := valMap[t]; ok {
				ySeries[i] = opts.LineData{Value: val}
			} else {
				ySeries[i] = opts.LineData{Value: nil} // Lücke in der Zeitreihe
			}
		}
		line.AddSeries(series.LocationName, ySeries)
	}

	SetCommonSeriesOptions(line)
	return RenderToFile(line, outputPath)
}
