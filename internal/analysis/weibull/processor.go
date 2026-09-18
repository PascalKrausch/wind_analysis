package weibull

import (
	"fmt"
	"math"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/statistics"
	"wind_analysis/internal/analysis/visualization"
	"wind_analysis/models"
)

type windSpeedGetter func(models.WindData) *float64

type heightSpec struct {
	heightM int
	getter  windSpeedGetter
}

var defaultHeightSpecs = []heightSpec{
	{heightM: 10, getter: func(d models.WindData) *float64 { return d.WindSpeed_10m }},
	{heightM: 80, getter: func(d models.WindData) *float64 { return d.WindSpeed_80m }},
	{heightM: 100, getter: func(d models.WindData) *float64 { return d.WindSpeed_100m }},
	{heightM: 120, getter: func(d models.WindData) *float64 { return d.WindSpeed_120m }},
	{heightM: 180, getter: func(d models.WindData) *float64 { return d.WindSpeed_180m }},
	{heightM: 200, getter: func(d models.WindData) *float64 { return d.WindSpeed_200m }},
}

// BuildPlotsForLocation erstellt Weibull-Plot-Inputs für alle Höhen eines Standorts
func BuildPlotsForLocation(locationName string, records []models.WindRecord) ([]visualization.WeibullPlotInput, []models.WeibullAnalysisResult, error) {
	inputs := make([]visualization.WeibullPlotInput, 0, len(defaultHeightSpecs))
	results := make([]models.WeibullAnalysisResult, 0, len(defaultHeightSpecs))

	for _, spec := range defaultHeightSpecs {
		values := extractWindSpeedSeries(records, spec.getter)
		values = statistics.FilterFinite(values)
		values = filterNonNegative(values)
		
		if len(values) < 3 {
			continue
		}

		analysis, err := distribution.AnalyzeWeibullSeries(
			fmt.Sprintf("%s @ %dm", locationName, spec.heightM), 
			spec.heightM, 
			values,
		)
		if err != nil {
			continue
		}

		modelResult := toModelWeibullResult(analysis)
		sorted := statistics.SortedCopy(values)
		stats := statistics.CalcCoreStats(sorted)
		histogram := statistics.CalcHistogram(sorted, chooseHistogramBinCount(len(sorted)), stats)
		empiricalCDF := statistics.CalcCumulativeDistribution(sorted)
		
		fittedPDF, fittedCDF := buildWeibullCurves(sorted, distribution.WeibullParams{
			Shape:    analysis.Shape,
			Scale:    analysis.Scale,
			Location: analysis.Location,
		}, 128)

		inputs = append(inputs, visualization.WeibullPlotInput{
			SeriesName:   modelResult.SeriesName,
			LocationName: locationName,
			HeightM:      spec.heightM,
			Analysis:     modelResult,
			Histogram:    histogram,
			EmpiricalCDF: empiricalCDF,
			FittedPDF:    fittedPDF,
			FittedCDF:    fittedCDF,
		})
		results = append(results, modelResult)
	}

	if len(inputs) == 0 {
		return nil, nil, fmt.Errorf("keine gültigen Weibull-Serien für %s gefunden", locationName)
	}

	return inputs, results, nil
}

// BuildComparisons erstellt Vergleichsergebnisse zwischen allen Weibull-Serien
func BuildComparisons(results []models.WeibullAnalysisResult) []models.WeibullComparisonResult {
	if len(results) < 2 {
		return nil
	}

	out := make([]models.WeibullComparisonResult, 0, len(results)*(len(results)-1)/2)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			left := toDistributionWeibullResult(results[i])
			right := toDistributionWeibullResult(results[j])
			comparison := distribution.CompareWeibullSeries(left, right)
			out = append(out, toModelWeibullComparisonResult(comparison))
		}
	}
	return out
}

// toDistributionWeibullResult konvertiert ein models.Result in ein distribution.Result
func toDistributionWeibullResult(in models.WeibullAnalysisResult) distribution.WeibullAnalysisResult {
	return distribution.WeibullAnalysisResult{
		SeriesName:    in.SeriesName,
		HeightM:       in.HeightM,
		SampleSize:    in.SampleSize,
		Shape:         in.Shape,
		Scale:         in.Scale,
		Location:      in.Location,
		LogLikelihood: in.LogLikelihood,
		AIC:           in.AIC,
		BIC:           in.BIC,
		KSStatistic:   in.KSStatistic,
		RMSE:          in.RMSE,
	}
}

// toModelWeibullComparisonResult konvertiert ein distribution.ComparisonResult in ein models.ComparisonResult
func toModelWeibullComparisonResult(in distribution.WeibullComparisonResult) models.WeibullComparisonResult {
	return models.WeibullComparisonResult{
		LeftSeriesName:  in.LeftSeriesName,
		RightSeriesName: in.RightSeriesName,
		LeftHeightM:     in.LeftHeightM,
		RightHeightM:    in.RightHeightM,
		ShapeDelta:      in.ShapeDelta,
		ScaleDelta:      in.ScaleDelta,
		LocationDelta:   in.LocationDelta,
		KSStatistic:     in.KSStatistic,
		MeanAbsCDFDiff:  in.MeanAbsCDFDiff,
	}
}

// extractWindSpeedSeries extrahiert Windgeschwindigkeiten für eine bestimmte Höhe
func extractWindSpeedSeries(records []models.WindRecord, getter windSpeedGetter) []float64 {
	out := make([]float64, 0, len(records))
	for _, record := range records {
		if v := getter(record.WindData); v != nil {
			out = append(out, *v)
		}
	}
	return out
}

// filterNonNegative entfernt negative Werte
func filterNonNegative(data []float64) []float64 {
	out := make([]float64, 0, len(data))
	for _, v := range data {
		if v >= 0 {
			out = append(out, v)
		}
	}
	return out
}

// chooseHistogramBinCount wählt eine angemessene Anzahl von Histogramm-Bins
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

// buildWeibullCurves berechnet PDF und CDF Kurven für eine Weibull-Verteilung
func buildWeibullCurves(sortedData []float64, params distribution.WeibullParams, pointsCount int) ([]statistics.DensityPoint, []statistics.CDFPoint) {
	if len(sortedData) == 0 || pointsCount < 2 || !params.Valid() {
		return nil, nil
	}

	stats := statistics.CalcCoreStats(sortedData)
	q1 := statistics.CalcPercentileFraction(sortedData, 0.25)
	q3 := statistics.CalcPercentileFraction(sortedData, 0.75)
	spread := math.Max(q3-q1, stats.StdDev)
	if spread <= 0 {
		spread = math.Max(params.Scale, 1)
	}

	minX := math.Max(0, math.Min(sortedData[0], params.Location)-spread)
	maxX := math.Max(sortedData[len(sortedData)-1], params.Location+3*params.Scale) + spread
	if maxX <= minX {
		maxX = minX + 1
	}

	step := (maxX - minX) / float64(pointsCount-1)
	pdf := make([]statistics.DensityPoint, pointsCount)
	cdf := make([]statistics.CDFPoint, pointsCount)

	for i := 0; i < pointsCount; i++ {
		x := minX + float64(i)*step
		pdf[i] = statistics.DensityPoint{X: x, Y: distribution.WeibullPDF(x, params)}
		cdf[i] = statistics.CDFPoint{X: x, Y: distribution.WeibullCDF(x, params)}
	}

	return pdf, cdf
}

// toModelWeibullResult konvertiert ein distribution.Result in ein models.Result
func toModelWeibullResult(in distribution.WeibullAnalysisResult) models.WeibullAnalysisResult {
	return models.WeibullAnalysisResult{
		SeriesName:    in.SeriesName,
		HeightM:       in.HeightM,
		SampleSize:    in.SampleSize,
		Shape:         in.Shape,
		Scale:         in.Scale,
		Location:      in.Location,
		LogLikelihood: in.LogLikelihood,
		AIC:           in.AIC,
		BIC:           in.BIC,
		KSStatistic:   in.KSStatistic,
		RMSE:          in.RMSE,
	}
}