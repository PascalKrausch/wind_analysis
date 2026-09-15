package statistics

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/distuv"
)

// Bin repräsentiert eine Klasse im Histogramm
type Bin struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

// DensityPoint repräsentiert einen Punkt auf der kontinuierlichen Dichtekurve
type DensityPoint struct {
	X float64 `json:"x"` // Der physikalische Wert (z.B. Temperatur)
	Y float64 `json:"y"` // Die Wahrscheinlichkeitsdichte an dieser Stelle
}

// JarqueBeraResult hält die Testergebnisse für den Normalverteilungstest
type JarqueBeraResult struct {
	Statistic float64 `json:"statistic"` // Prüfzahl (Je höher, desto unähnlicher zur Normalverteilung)
	IsNormal  bool    `json:"is_normal"` // true, wenn die Daten statistisch als normalverteilt gelten (95% Niveau)
}

// CDFPoint repräsentiert einen Punkt auf der kumulativen Dichtekurve
type CDFPoint struct {
	X float64 `json:"x"` // Der physikalische Wert
	Y float64 `json:"y"` // Die kumulierte Wahrscheinlichkeit (0.0 bis 1.0)
}

// WeibullResult hält die charakteristischen Parameter einer Windverteilung
type WeibullResult struct {
	Shape float64 `json:"shape"` // k-Faktor (Form): Zeigt an, wie gleichmäßig der Wind weht (meist 1.5 - 3.0)
	Scale float64 `json:"scale"` // c-Faktor (Skalierung): Entspricht in etwa der mittleren Windgeschwindigkeit in m/s
}

// Bündelung der Deskriptiven Statistik
type CoreStats struct {
	Mean   float64
	StdDev float64
	Var    float64
	Min    float64
	Max    float64
}

// CalcCoreStats berechnet Mean, Variance, StdDev, Min & Max in max. 2 Durchläufen
func CalcCoreStats(data []float64) CoreStats {
	n := len(data)
	if n < 2 {
		return CoreStats{}
	}

	variance := stat.Variance(data, nil) // Stichprobenvarianz (n-1), wie bisher
	return CoreStats{
		Mean:   stat.Mean(data, nil),
		Var:    variance,
		StdDev: math.Sqrt(variance),
		Min:    floats.Min(data),
		Max:    floats.Max(data),
	}
}

// =========================================================================
// 1. Skewness (Schiefe)
// =========================================================================

// CalcSkewness berechnet die Schiefe der Verteilung (Fisher-Pearson-Koeffizient).
// Ein Wert von 0 bedeutet perfekte Symmetrie.
// Positive Werte = Rechtsschief
// Negative Werte = Linksschief
func CalcSkewness(data []float64, stats CoreStats) float64 {
	if len(data) < 3 || stats.StdDev == 0 {
		return 0
	}
	return stat.Skew(data, nil)
}

// =========================================================================
// 2. Kurtosis (Wölbung / Steilheit)
// =========================================================================

// CalcKurtosis berechnet die Excess-Kurtosis (Wölbung relativ zur Normalverteilung).
// Eine Normalverteilung hat eine Excess-Kurtosis von exakt 0.
// Werte > 0 (leptokurtisch): Spitzere Kurve, dicke Enden -> Hohe Wahrscheinlichkeit für extreme Ausreißer.
// Werte < 0 (platykurtisch): Flachere Kurve, dünne Enden -> Werte konzentrieren sich eng um das Mittel.
func CalcKurtosis(data []float64, stats CoreStats) float64 {
	if len(data) < 4 || stats.StdDev == 0 {
		return 0
	}
	return stat.ExKurtosis(data, nil)
}

// =========================================================================
// 3. Histogram
// =========================================================================

// CalcHistogram unterteilt die Daten in eine definierte Anzahl von Klassen (Bins)
// und zählt die Häufigkeiten.
func CalcHistogram(data []float64, binCount int, stats CoreStats) []Bin {
	n := len(data)
	if n == 0 || binCount <= 0 {
		return nil
	}

	min := stats.Min
	max := stats.Max
	if min == max {
		return []Bin{{Min: min, Max: max, Count: n}}
	}

	binWidth := (max - min) / float64(binCount)
	dividers := make([]float64, binCount+1)
	for i := 0; i <= binCount; i++ {
		dividers[i] = min + float64(i)*binWidth
	}
	dividers[binCount] = max // numerische Stabilität am rechten Rand

	counts := make([]float64, binCount)
	stat.Histogram(counts, dividers, data, nil)

	bins := make([]Bin, binCount)
	for i := 0; i < binCount; i++ {
		bins[i] = Bin{
			Min:   dividers[i],
			Max:   dividers[i+1],
			Count: int(math.Round(counts[i])),
		}
	}
	return bins
}

// =========================================================================
// 4. Density (Kernel Density Estimation - KDE)
// =========================================================================

// CalcDensity berechnet die Wahrscheinlichkeitsdichte mittels eines Gauß-Kernels.

func CalcDensity(sortedData []float64, stats CoreStats, iqr float64, pointsCount int) []DensityPoint {
	n := len(sortedData)
	if n < 2 || pointsCount <= 0 {
		return nil
	}

	min := sortedData[0]
	max := sortedData[n-1]

	h := 0.9 * stats.StdDev * math.Pow(float64(n), -0.2)
	if iqr > 0 {
		h = 0.9 * math.Min(stats.StdDev, iqr/1.34) * math.Pow(float64(n), -0.2)
	}
	if h <= 0 {
		h = 1e-5
	}

	rangeMin := min - (3.0 * h)
	rangeMax := max + (3.0 * h)
	step := (rangeMax - rangeMin) / float64(pointsCount-1)

	results := make([]DensityPoint, pointsCount)

	invNH := 1.0 / (float64(n) * h)
	cutoff := 4.0 * h
	left, right := 0, 0

	for i := 0; i < pointsCount; i++ {
		x := rangeMin + float64(i)*step
		low := x - cutoff
		high := x + cutoff

		for left < n && sortedData[left] < low {
			left++
		}
		for right < n && sortedData[right] <= high {
			right++
		}

		sum := 0.0
		for j := left; j < right; j++ {
			u := (x - sortedData[j]) / h
			sum += distuv.UnitNormal.Prob(u)
		}

		results[i] = DensityPoint{X: x, Y: sum * invNH}
	}

	return results
}

// =========================================================================
// 5. Theoretische Gauß-Verteilung (Normalverteilung)
// =========================================================================

// CalcGaussianDistribution berechnet die ideale Normalverteilungskurve
// basierend auf Mittelwert und Standardabweichung der echten Daten.
// Formel: f(x) = (1 / (sigma * sqrt(2*pi))) * e^(-0.5 * ((x-mu)/sigma)^2)
func CalcGaussianDistribution(data []float64, stats CoreStats, pointsCount int) []DensityPoint {
	n := len(data)
	if n < 2 || pointsCount <= 0 || stats.StdDev <= 0 {
		return nil
	}

	min := stats.Mean - (3.0 * stats.StdDev)
	max := stats.Mean + (3.0 * stats.StdDev)
	step := (max - min) / float64(pointsCount-1)

	normal := distuv.Normal{Mu: stats.Mean, Sigma: stats.StdDev}
	results := make([]DensityPoint, pointsCount)
	for i := 0; i < pointsCount; i++ {
		x := min + float64(i)*step
		results[i] = DensityPoint{X: x, Y: normal.Prob(x)}
	}

	return results
}

// =========================================================================
// 6. Jarque-Bera-Test (Prüfung auf Normalverteilung)
// =========================================================================

// CalcJarqueBeraTest prüft, ob die Datenmenge als normalverteilt angenommen werden kann.
// Formel: JB = (n / 6) * (S^2 + (C^2 / 4)) wobei S = Skewness und C = Excess-Kurtosis.
func CalcJarqueBeraTest(data []float64, stats CoreStats) JarqueBeraResult {
	n := len(data)
	if n < 4 {
		return JarqueBeraResult{}
	}

	skew := CalcSkewness(data, stats)
	kurt := CalcKurtosis(data, stats)
	jb := (float64(n) / 6.0) * ((skew * skew) + ((kurt * kurt) / 4.0))

	critical95 := distuv.ChiSquared{K: 2}.Quantile(0.95)

	return JarqueBeraResult{
		Statistic: jb,
		IsNormal:  jb < critical95,
	}
}

// =========================================================================
// 7. Empirische Kumulative Verteilungsfunktion (eCDF)
// =========================================================================

// CalcCumulativeDistribution berechnet die eCDF.
// Sie zeigt für jeden Wert, wie viel Prozent der Daten kleiner oder gleich diesem Wert sind.
func CalcCumulativeDistribution(data []float64) []CDFPoint {
	n := len(data)
	if n == 0 {
		return nil
	}

	// Wir arbeiten auf einer Kopie, um die Originaldaten nicht zu verändern
	sorted := make([]float64, n)
	copy(sorted, data)
	sort.Float64s(sorted)

	results := make([]CDFPoint, n)
	for i, v := range sorted {
		results[i] = CDFPoint{
			X: v,
			Y: float64(i+1) / float64(n),
		}
	}

	return results
}

// =========================================================================
// 8. Weibull-Parameter-Schätzung (für Windkraft)
// =========================================================================

// EstimateWeibullParameters schätzt die Parameter k (Form) und c (Skalierung)
// für Windgeschwindigkeiten mithilfe der Justus-Approximation.
func EstimateWeibullParameters(data []float64, stats CoreStats) WeibullResult {
	n := len(data)
	if n < 2 {
		return WeibullResult{}
	}

	if stats.Mean <= 0 || stats.StdDev <= 0 {
		return WeibullResult{}
	}

	// Justus-Approximation für den Formparameter k:
	// k = (stdDev / mean)^(-1.086)
	shape := math.Pow(stats.StdDev/stats.Mean, -1.086)

	// Skalenparameter c berechnen:
	// c = mean / Gamma(1 + 1/k)
	scale := stats.Mean / math.Gamma(1.0+(1.0/shape))

	return WeibullResult{
		Shape: shape,
		Scale: scale,
	}
}
