package fitting

import (
	"fmt"
	"math"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/statistics"
	"wind_analysis/models"
)

type ValueGetter func(models.WindData) *float64

type HeightSpec struct {
	HeightM int
	Getter  ValueGetter
}

var DefaultHeightSpecs = []HeightSpec{
	{HeightM: 10, Getter: func(d models.WindData) *float64 { return d.WindSpeed_10m }},
	{HeightM: 80, Getter: func(d models.WindData) *float64 { return d.WindSpeed_80m }},
	{HeightM: 100, Getter: func(d models.WindData) *float64 { return d.WindSpeed_100m }},
	{HeightM: 120, Getter: func(d models.WindData) *float64 { return d.WindSpeed_120m }},
	{HeightM: 180, Getter: func(d models.WindData) *float64 { return d.WindSpeed_180m }},
	{HeightM: 200, Getter: func(d models.WindData) *float64 { return d.WindSpeed_200m }},
}

// AnalysisPlotInput bündelt Plot-Daten unabhängig von der konkreten Verteilung.
type AnalysisPlotInput struct {
	SeriesName   string
	LocationName string
	HeightM      int
	FitterName   string
	Model        distribution.ContinuousDistribution
	Metrics      distribution.FitMetrics
	Histogram    []statistics.Bin
	EmpiricalCDF []statistics.CDFPoint
	FittedPDF    []statistics.DensityPoint
	FittedCDF    []statistics.CDFPoint
}

// BuildPlotsForLocation passt eine spezifische Verteilung (via fitter) über alle Höhen an.
func BuildPlotsForLocation(
	locationName string,
	records []models.WindRecord,
	fitter distribution.Fitter,
	specs []HeightSpec,
) ([]AnalysisPlotInput, error) {
	if len(specs) == 0 {
		specs = DefaultHeightSpecs
	}

	inputs := make([]AnalysisPlotInput, 0, len(specs))

	for _, spec := range specs {
		values := extractSeries(records, spec.Getter)
		cleaned := distribution.CleanData(values)

		if len(cleaned) < 3 {
			continue
		}

		// 1. Generischer Fit
		model, err := fitter.Fit(cleaned)
		if err != nil {
			continue
		}

		// 2. Generische Evaluation
		metrics, err := distribution.EvaluateFit(model, cleaned)
		if err != nil {
			continue
		}

		// 3. Kurven & Diagramme aufbauen
		sorted := distribution.SortedCopy(cleaned)
		stats := statistics.CalcCoreStats(sorted)
		histogram := statistics.CalcHistogram(sorted, chooseHistogramBinCount(len(sorted)), stats)
		empiricalCDF := statistics.CalcCumulativeDistribution(sorted)

		fittedPDF, fittedCDF := buildDistributionCurves(sorted, model, 128)

		inputs = append(inputs, AnalysisPlotInput{
			SeriesName:   fmt.Sprintf("%s @ %dm (%s)", locationName, spec.HeightM, fitter.Name()),
			LocationName: locationName,
			HeightM:      spec.HeightM,
			FitterName:   fitter.Name(),
			Model:        model,
			Metrics:      metrics,
			Histogram:    histogram,
			EmpiricalCDF: empiricalCDF,
			FittedPDF:    fittedPDF,
			FittedCDF:    fittedCDF,
		})
	}

	if len(inputs) == 0 {
		return nil, fmt.Errorf("keine gültigen Fits für Standort %s mit %s gefunden", locationName, fitter.Name())
	}

	return inputs, nil
}

// FindBestFitsForLocation ermittelt für jede Höhe automatisch die am besten passende Verteilung (z. B. via AIC).
func FindBestFitsForLocation(
	locationName string,
	records []models.WindRecord,
	fitters []distribution.Fitter,
	specs []HeightSpec,
) ([]AnalysisPlotInput, error) {
	if len(specs) == 0 {
		specs = DefaultHeightSpecs
	}

	results := make([]AnalysisPlotInput, 0, len(specs))

	for _, spec := range specs {
		values := extractSeries(records, spec.Getter)
		cleaned := distribution.CleanData(values)
		if len(cleaned) < 3 {
			continue
		}

		var bestInput *AnalysisPlotInput
		bestAIC := math.Inf(1)

		// Teste alle angegebenen Fitter
		for _, fitter := range fitters {
			model, err := fitter.Fit(cleaned)
			if err != nil {
				continue
			}

			metrics, err := distribution.EvaluateFit(model, cleaned)
			if err != nil {
				continue
			}

			// Kriterium: Niedrigster AIC gewinnt
			if metrics.AIC < bestAIC {
				bestAIC = metrics.AIC

				sorted := distribution.SortedCopy(cleaned)
				stats := statistics.CalcCoreStats(sorted)
				histogram := statistics.CalcHistogram(sorted, chooseHistogramBinCount(len(sorted)), stats)
				empiricalCDF := statistics.CalcCumulativeDistribution(sorted)
				fittedPDF, fittedCDF := buildDistributionCurves(sorted, model, 128)

				bestInput = &AnalysisPlotInput{
					SeriesName:   fmt.Sprintf("%s @ %dm", locationName, spec.HeightM),
					LocationName: locationName,
					HeightM:      spec.HeightM,
					FitterName:   fitter.Name(),
					Model:        model,
					Metrics:      metrics,
					Histogram:    histogram,
					EmpiricalCDF: empiricalCDF,
					FittedPDF:    fittedPDF,
					FittedCDF:    fittedCDF,
				}
			}
		}

		if bestInput != nil {
			results = append(results, *bestInput)
		}
	}

	return results, nil
}

// Generische Berechnung von PDF/CDF Kurvenpunkte über das ContinuousDistribution-Interface
func buildDistributionCurves(
	sortedData []float64,
	dist distribution.ContinuousDistribution,
	pointsCount int,
) ([]statistics.DensityPoint, []statistics.CDFPoint) {
	if len(sortedData) == 0 || pointsCount < 2 || dist == nil {
		return nil, nil
	}

	minX := math.Max(0, sortedData[0]*0.8)
	maxX := sortedData[len(sortedData)-1] * 1.2
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

func extractSeries(records []models.WindRecord, getter ValueGetter) []float64 {
	out := make([]float64, 0, len(records))
	for _, record := range records {
		if v := getter(record.WindData); v != nil {
			out = append(out, *v)
		}
	}
	return out
}

func chooseHistogramBinCount(n int) int {
	if n <= 0 {
		return 1
	}
	bins := int(math.Round(math.Sqrt(float64(n))))
	if bins < 6 {
		bins = 6
	}
	if bins > 24 {
		bins = 24
	}
	return bins
}
