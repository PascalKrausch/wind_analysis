package visualization

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"

	"wind_analysis/internal/analysis/interpolation"
)

// PlotHellmannExponentTimeline visualisiert Hellmann-Exponent über Zeit als Line-Chart.
func PlotHellmannExponentTimeline(exponents []interpolation.HellmannExponentResult, outputPath string) error {
	if len(exponents) == 0 {
		return fmt.Errorf("keine Hellmann-Exponenten für Timeline vorhanden")
	}

	type agg struct {
		sum   float64
		count int
	}

	byTime := make(map[time.Time]agg)
	for _, exp := range exponents {
		if math.IsNaN(exp.Alpha) {
			continue
		}
		a := byTime[exp.Time]
		a.sum += exp.Alpha
		a.count++
		byTime[exp.Time] = a
	}

	if len(byTime) == 0 {
		return fmt.Errorf("keine gültigen Hellmann-Exponenten für Timeline")
	}

	times := make([]time.Time, 0, len(byTime))
	for t := range byTime {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	xData := ConvertTimesToXAxisHourly(times)
	yData := make([]float64, len(times))
	for i, t := range times {
		a := byTime[t]
		yData[i] = a.sum / float64(a.count)
	}

	cfg := DefaultChartConfig()
	cfg.Title = "Hellmann-Exponent über Zeit"
	cfg.Subtitle = "Mittelwert über alle Standorte"
	cfg.XAxisName = "Zeit"
	cfg.YAxisName = "Hellmann-Exponent α"
	cfg.EnableDataZoom = true

	line := CreateLineChart(cfg)
	AddLineSeries(line, "α (Mittelwert)", xData, yData)
	SetCommonSeriesOptions(line)

	return RenderToFile(line, outputPath)
}

// PlotModelValidation ist bewusst deaktiviert (nur noch Tabellenübersichten).
func PlotModelValidation(stats interpolation.ValidationStats, height float64, outputPath string) error {
	return fmt.Errorf("scatter-plot deaktiviert: bitte tabellarische Übersichten verwenden")
}

// PlotValidationMetrics erzeugt eine Tabelle (abwärtskompatibel: ohne Ortsdimension => 'Gesamt').
func PlotValidationMetrics(results map[float64]interpolation.ValidationResult, outputPath string) error {
	return PlotValidationMetricsByHeightAndLocation(
		map[string]map[float64]interpolation.ValidationResult{"Gesamt": results},
		outputPath,
	)
}

// PlotValidationMetricsByHeightAndLocation erzeugt eine tabellarische Übersicht nach Ort und Höhe.
func PlotValidationMetricsByHeightAndLocation(results map[string]map[float64]interpolation.ValidationResult, outputPath string) error {
	if len(results) == 0 {
		return fmt.Errorf("keine Validierungsergebnisse vorhanden")
	}

	locations := make([]string, 0, len(results))
	for loc := range results {
		locations = append(locations, loc)
	}
	sort.Strings(locations)

	rows := make([][]string, 0)
	for _, loc := range locations {
		heightMap := results[loc]
		heights := make([]float64, 0, len(heightMap))
		for h := range heightMap {
			heights = append(heights, h)
		}
		sort.Float64s(heights)

		for _, h := range heights {
			r := heightMap[h]
			rows = append(rows, []string{
				loc,
				fmt.Sprintf("%.0f", h),
				fmt.Sprintf("%.4f", r.MAE),
				fmt.Sprintf("%.4f", r.RMSE),
				fmt.Sprintf("%.4f", r.Correlation),
				fmt.Sprintf("%d", r.SampleCount),
			})
		}
	}

	return RenderTable(
		outputPath,
		"Validierungsmetriken nach Höhe und Ort",
		"Power-Law-Modell (MAE, RMSE, Korrelation)",
		[]string{"Ort", "Höhe [m]", "MAE", "RMSE", "Korrelation", "n"},
		rows,
	)
}

// PlotMultiLocationComparison vergleicht Alpha-Werte über mehrere Standorte als Chart.
func PlotMultiLocationComparison(exponents []interpolation.HellmannExponentResult, outputPath string) error {
	if len(exponents) == 0 {
		return fmt.Errorf("keine Hellmann-Exponenten für Standortvergleich vorhanden")
	}

	type agg struct {
		sum   float64
		count int
	}

	locationTime := make(map[string]map[time.Time]agg)
	allTimesSet := make(map[time.Time]struct{})

	for _, exp := range exponents {
		if math.IsNaN(exp.Alpha) {
			continue
		}
		loc := exp.Location.Name
		if loc == "" {
			loc = "Unbekannt"
		}
		if _, ok := locationTime[loc]; !ok {
			locationTime[loc] = make(map[time.Time]agg)
		}
		a := locationTime[loc][exp.Time]
		a.sum += exp.Alpha
		a.count++
		locationTime[loc][exp.Time] = a
		allTimesSet[exp.Time] = struct{}{}
	}

	if len(allTimesSet) == 0 || len(locationTime) == 0 {
		return fmt.Errorf("keine gültigen Daten für Standortvergleich")
	}

	times := make([]time.Time, 0, len(allTimesSet))
	for t := range allTimesSet {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

	locationNames := make([]string, 0, len(locationTime))
	for loc := range locationTime {
		locationNames = append(locationNames, loc)
	}
	sort.Strings(locationNames)

	cfg := DefaultChartConfig()
	cfg.Title = "Hellmann-Exponent: Standortvergleich"
	cfg.XAxisName = "Zeit"
	cfg.YAxisName = "Hellmann-Exponent α"
	cfg.EnableDataZoom = true

	line := CreateLineChart(cfg)
	line.SetXAxis(ConvertTimesToXAxisHourly(times))

	for _, loc := range locationNames {
		series := make([]opts.LineData, len(times))
		for i, t := range times {
			if a, ok := locationTime[loc][t]; ok && a.count > 0 {
				series[i] = opts.LineData{Value: a.sum / float64(a.count)}
			} else {
				series[i] = opts.LineData{Value: nil}
			}
		}
		line.AddSeries(loc, series)
	}

	SetCommonSeriesOptions(line)
	return RenderToFile(line, outputPath)
}

// PlotValidationDescriptiveStatsTable zeigt deskriptive Statistik je Ort/Höhe als Tabelle.
func PlotValidationDescriptiveStatsTable(results map[string]map[float64]interpolation.ValidationResult, outputPath string) error {
	if len(results) == 0 {
		return fmt.Errorf("keine Validierungsergebnisse vorhanden")
	}

	locations := make([]string, 0, len(results))
	for loc := range results {
		locations = append(locations, loc)
	}
	sort.Strings(locations)

	rows := make([][]string, 0)
	for _, loc := range locations {
		heightMap := results[loc]
		heights := make([]float64, 0, len(heightMap))
		for h := range heightMap {
			heights = append(heights, h)
		}
		sort.Float64s(heights)

		for _, h := range heights {
			r := heightMap[h]
			rows = append(rows, []string{
				loc,
				fmt.Sprintf("%.0f", h),
				fmt.Sprintf("%d", r.SampleCount),

				fmt.Sprintf("%.4f", r.PredictedSummary.Mean),
				fmt.Sprintf("%.4f", r.PredictedSummary.Median),
				fmt.Sprintf("%.4f", r.PredictedSummary.StandardDeviation),
				fmt.Sprintf("%.4f", r.PredictedSummary.Variance),
				fmt.Sprintf("%.4f", r.PredictedSummary.CoefficientOfVariation),
				fmt.Sprintf("%.4f", r.PredictedSummary.Skewness),
				fmt.Sprintf("%.4f", r.PredictedSummary.Kurtosis),

				fmt.Sprintf("%.4f", r.ActualSummary.Mean),
				fmt.Sprintf("%.4f", r.ActualSummary.Median),
				fmt.Sprintf("%.4f", r.ActualSummary.StandardDeviation),
				fmt.Sprintf("%.4f", r.ActualSummary.Variance),
				fmt.Sprintf("%.4f", r.ActualSummary.CoefficientOfVariation),
				fmt.Sprintf("%.4f", r.ActualSummary.Skewness),
				fmt.Sprintf("%.4f", r.ActualSummary.Kurtosis),

				fmt.Sprintf("%.4f", r.ErrorSummary.Mean),
				fmt.Sprintf("%.4f", r.ErrorSummary.Median),
				fmt.Sprintf("%.4f", r.ErrorSummary.StandardDeviation),
				fmt.Sprintf("%.4f", r.ErrorSummary.Variance),
				fmt.Sprintf("%.4f", r.ErrorSummary.CoefficientOfVariation),
				fmt.Sprintf("%.4f", r.ErrorSummary.Skewness),
				fmt.Sprintf("%.4f", r.ErrorSummary.Kurtosis),
			})
		}
	}

	return RenderTable(
		outputPath,
		"Deskriptive Statistik nach Ort und Höhe",
		"Power-Law-Validierung (Predicted / Actual / Error)",
		[]string{
			"Ort", "Höhe [m]", "n",
			"Pred Mean", "Pred Median", "Pred StdDev", "Pred Var", "Pred CV", "Pred Skew", "Pred Kurt",
			"Act Mean", "Act Median", "Act StdDev", "Act Var", "Act CV", "Act Skew", "Act Kurt",
			"Err Mean", "Err Median", "Err StdDev", "Err Var", "Err CV", "Err Skew", "Err Kurt",
		},
		rows,
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
