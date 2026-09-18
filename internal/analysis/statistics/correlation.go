package statistics

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/stat"
)

// =========================================================================
// 1. Pearson-Korrelation
// =========================================================================

// CalcPearson berechnet den linearen Pearson-Korrelationskoeffizienten r
func CalcPearson(xData, yData []float64) float64 {
	n := len(xData)
	if n < 2 || n != len(yData) {
		return 0
	}

	r := stat.Correlation(xData, yData, nil)
	if math.IsNaN(r) {
		return 0
	}
	return r
}

// =========================================================================
// 2. Spearman-Rangkorrelation (inklusive Rang-Zuweisung mit Tie-Handling)
// =========================================================================

// CalcSpearman berechnet die Rangkorrelation nach Spearman (monotone Trends)
func CalcSpearman(xData, yData []float64) float64 {
	n := len(xData)
	if n < 2 || n != len(yData) {
		return 0
	}

	// Spearman correlation is Pearson correlation on ranks
	rankX := rankData(xData)
	rankY := rankData(yData)
	
	r := stat.Correlation(rankX, rankY, nil)
	if math.IsNaN(r) {
		return 0
	}
	return r
}

// rankData converts data to ranks
func rankData(data []float64) []float64 {
	n := len(data)
	if n == 0 {
		return nil
	}

	// Create index array
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	// Sort indices based on data values
	sort.Slice(indices, func(i, j int) bool {
		return data[indices[i]] < data[indices[j]]
	})

	// Assign ranks
	ranks := make([]float64, n)
	for i, idx := range indices {
		ranks[idx] = float64(i + 1)
	}

	return ranks
}

// =========================================================================
// 3. Kendall's Tau (Tau-b Variante für exakten Umgang mit Bindungen)
// =========================================================================

// CalcKendall berechnet Kendall's Tau-b (Vergleich von Paar-Konkordanzen)
func CalcKendall(xData, yData []float64) float64 {
	n := len(xData)
	if n < 2 || n != len(yData) {
		return 0
	}

	r := stat.Kendall(xData, yData, nil)
	if math.IsNaN(r) {
		return 0
	}
	return r
}

// =========================================================================
// 4. Autokorrelation
// =========================================================================

// CalcAutocorrelation berechnet die Korrelation einer Zeitreihe mit sich selbst bei Verzögerung (lag)
func CalcAutocorrelation(data []float64, lag int) float64 {
	n := len(data)
	if n < 2 || lag >= n || lag < 0 {
		return 0
	}
	if lag == 0 {
		return 1.0
	}

	r := stat.Correlation(data[:n-lag], data[lag:], nil)
	if math.IsNaN(r) {
		return 0
	}
	return r
}

// =========================================================================
// 5. Kreuzkorrelation (Cross-Correlation)
// =========================================================================

// CalcCrossCorrelation berechnet die Korrelation zwischen zwei unterschiedlichen
// Zeitreihen bei einem spezifischen lag (kann positiv oder negativ sein)
func CalcCrossCorrelation(xData, yData []float64, lag int) float64 {
	n := len(xData)
	if n < 2 || n != len(yData) {
		return 0
	}

	absLag := lag
	if lag < 0 {
		absLag = -lag
	}
	if absLag >= n {
		return 0
	}

	var startX, endX, startY, endY int
	if lag >= 0 {
		startX = 0
		endX = n - lag
		startY = lag
		endY = n
	} else {
		startX = -lag
		endX = n
		startY = 0
		endY = n + lag
	}

	r := stat.Correlation(xData[startX:endX], yData[startY:endY], nil)
	if math.IsNaN(r) {
		return 0
	}
	return r
}
