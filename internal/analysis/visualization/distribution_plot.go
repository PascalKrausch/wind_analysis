package visualization

import (
	"fmt"
	"math"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/statistics"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// DistributionPlotInput enthält die Daten für die Visualisierung einer beliebigen Verteilung.
type DistributionPlotInput struct {
	SeriesName   string
	LocationName string
	HeightM      int

	// Das gefittete Modell
	Distribution distribution.ContinuousDistribution
	ModelName    string

	// Bereits berechnete Statistik-/Fit-Ergebnisse
	Metrics distribution.FitMetrics

	// Roh- oder Aufbereitungsdaten für Visualisierungen
	Histogram    []statistics.Bin
	EmpiricalCDF []statistics.CDFPoint
	FittedPDF    []statistics.DensityPoint
	FittedCDF    []statistics.CDFPoint
}

// BuildDistributionCurves berechnet PDF und CDF Kurven für eine beliebige ContinuousDistribution.
func BuildDistributionCurves(sortedData []float64, dist distribution.ContinuousDistribution, pointsCount int) ([]statistics.DensityPoint, []statistics.CDFPoint) {
	if len(sortedData) == 0 || pointsCount < 2 {
		return nil, nil
	}

	stats := statistics.CalcCoreStats(sortedData)
	q1 := statistics.CalcPercentileFraction(sortedData, 0.25)
	q3 := statistics.CalcPercentileFraction(sortedData, 0.75)
	spread := math.Max(q3-q1, stats.StdDev)
	if spread <= 0 {
		spread = 1.0
	}

	minX := math.Max(0, sortedData[0]-spread)
	maxX := sortedData[len(sortedData)-1] + spread
	if maxX <= minX {
		maxX = minX + 1
	}

	step := (maxX - minX) / float64(pointsCount-1)
	pdf := make([]statistics.DensityPoint, pointsCount)
	cdf := make([]statistics.CDFPoint, pointsCount)

	for i := 0; i < pointsCount; i++ {
		x := minX + float64(i)*step
		pdf[i] = statistics.DensityPoint{X: x, Y: dist.PDF(x)}
		cdf[i] = statistics.CDFPoint{X: x, Y: dist.CDF(x)}
	}

	return pdf, cdf
}

// NewDistributionDashboard erstellt ein Dashboard für eine beliebige Verteilung.
func NewDistributionDashboard(inputs []DistributionPlotInput) *components.Page {
	page := components.NewPage()
	page.PageTitle = "Distribution Analysis"

	for _, in := range inputs {
		page.AddCharts(
			BuildDistributionHistogramPDFChart(in),
			BuildDistributionCDFChart(in),
		)
	}

	if len(inputs) > 1 {
		page.AddCharts(BuildDistributionMetricsChart(inputs))
	}

	return page
}

// PlotDistributionDashboard speichert ein Distribution-Dashboard.
func PlotDistributionDashboard(inputs []DistributionPlotInput, outputPath string) error {
	return SavePage(NewDistributionDashboard(inputs), outputPath)
}

// DistributionChartTitle generiert einen Titel für Verteilungs-Charts.
func DistributionChartTitle(base string, in DistributionPlotInput) string {
	if in.LocationName == "" && in.HeightM == 0 {
		return base
	}
	return fmt.Sprintf("%s – %s @ %dm", base, in.LocationName, in.HeightM)
}

// DistributionShortLabel generiert eine kurze Bezeichnung für die Serie.
func DistributionShortLabel(in DistributionPlotInput) string {
	if in.LocationName == "" && in.SeriesName == "" {
		return fmt.Sprintf("%dm", in.HeightM)
	}
	if in.SeriesName != "" {
		return in.SeriesName
	}
	return fmt.Sprintf("%s @ %dm", in.LocationName, in.HeightM)
}

// BuildDistributionHistogramPDFChart erstellt ein Histogramm + PDF Chart für jede Verteilung.
func BuildDistributionHistogramPDFChart(in DistributionPlotInput) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    DistributionChartTitle(fmt.Sprintf("%s Histogramm + PDF", in.ModelName), in),
			Subtitle: "Histogramm (Anzahl) und Modell-PDF (Dichte) im selben Koordinatensystem",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Wert"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Anzahl"}),
	)
	bar.ExtendYAxis(opts.YAxis{Name: "Dichte"})

	x, centers := histogramXAxisAndCenters(in.Histogram)

	hist := make([]opts.BarData, 0, len(in.Histogram))
	for _, b := range in.Histogram {
		hist = append(hist, opts.BarData{Value: b.Count})
	}
	bar.SetXAxis(x).AddSeries("Histogramm", hist)

	pdf := densityAtXVals(in.FittedPDF, centers)
	line := charts.NewLine()
	line.SetXAxis(x).AddSeries(
		fmt.Sprintf("%s-PDF", in.ModelName),
		convertToLineData(pdf),
		charts.WithLineChartOpts(opts.LineChart{YAxisIndex: 1, Smooth: opts.Bool(true)}),
	)

	bar.Overlap(line)
	return bar
}

// BuildDistributionCDFChart erstellt ein CDF Chart für jede Verteilung.
func BuildDistributionCDFChart(in DistributionPlotInput) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    DistributionChartTitle(fmt.Sprintf("%s CDF", in.ModelName), in),
			Subtitle: "Empirische CDF vs. Modell-CDF",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Wert"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Kumulative Wahrscheinlichkeit", Min: 0, Max: 1}),
	)

	empX, empY := cdfSeries(in.EmpiricalCDF)
	modX, modY := cdfSeries(in.FittedCDF)
	x := mergeSortedXAxis(empX, modX)

	line.SetXAxis(x).
		AddSeries("Empirisch", convertToLineData(alignSeries(x, empX, empY))).
		AddSeries(fmt.Sprintf("%s-Modell", in.ModelName), convertToLineData(alignSeries(x, modX, modY)))

	return line
}

// BuildDistributionMetricsChart erstellt ein Metrics Chart für jede Verteilung.
func BuildDistributionMetricsChart(inputs []DistributionPlotInput) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Verteilungs Fit-Güte",
			Subtitle: "AIC, BIC, KS und RMSE je Serie",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Serie"}),
	)

	x := make([]string, 0, len(inputs))
	aic := make([]opts.LineData, 0, len(inputs))
	bic := make([]opts.LineData, 0, len(inputs))
	ks := make([]opts.LineData, 0, len(inputs))
	rmse := make([]opts.LineData, 0, len(inputs))

	for _, in := range inputs {
		x = append(x, DistributionShortLabel(in))
		aic = append(aic, opts.LineData{Value: in.Metrics.AIC})
		bic = append(bic, opts.LineData{Value: in.Metrics.BIC})
		ks = append(ks, opts.LineData{Value: in.Metrics.KSStatistic})
		rmse = append(rmse, opts.LineData{Value: in.Metrics.RMSE})
	}

	line.SetXAxis(x).
		AddSeries("AIC", aic).
		AddSeries("BIC", bic).
		AddSeries("KS", ks).
		AddSeries("RMSE", rmse)

	return line
}
