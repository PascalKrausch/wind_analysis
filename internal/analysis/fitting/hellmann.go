package fitting

import (
	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/interpolation"
	"wind_analysis/internal/analysis/statistics"
	"wind_analysis/models"
)

type HellmannFitResult struct {
	LocationName string
	Model        distribution.ContinuousDistribution
	Metrics      distribution.FitMetrics
	Histogram    []statistics.Bin
	EmpiricalCDF []statistics.CDFPoint
	FittedPDF    []statistics.DensityPoint
	FittedCDF    []statistics.CDFPoint
}

// FitHellmannDistribution passt Verteilung an Hellmann-Exponenten an
func FitHellmannDistribution(
	locationName string,
	records []models.WindRecord,
	fitters []distribution.Fitter,
) (HellmannFitResult, error) {
	// 1. Alpha-Werte extrahieren
	alphas := interpolation.ExtractAlphasFromRecords(records)

	// 2. Besten Fitter finden
	bestDist, bestMetrics, err := distribution.SelectBestModel(alphas)
	if err != nil {
		return HellmannFitResult{}, err
	}

	// 3. Plot-Daten generieren
	sorted := distribution.SortedCopy(alphas)
	stats := statistics.CalcCoreStats(sorted)
	histogram := statistics.CalcHistogram(sorted, chooseHistogramBinCount(len(sorted)), stats)
	empiricalCDF := statistics.CalcCumulativeDistribution(sorted)
	fittedPDF, fittedCDF := buildDistributionCurves(sorted, bestDist, 128)

	return HellmannFitResult{
		LocationName: locationName,
		Model:        bestDist,
		Metrics:      bestMetrics,
		Histogram:    histogram,
		EmpiricalCDF: empiricalCDF,
		FittedPDF:    fittedPDF,
		FittedCDF:    fittedCDF,
	}, nil
}
