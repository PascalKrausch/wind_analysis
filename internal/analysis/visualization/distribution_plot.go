package visualization

import (
	"fmt"
	"math"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/statistics"

	"github.com/go-echarts/go-echarts/v2/components"
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
