package statistics

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

// SummaryStats bündelt alle wichtigen statistischen Kennwerte auf einen Blick
type SummaryStats struct {
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Range    float64 `json:"range"`    // Spannweite (Max - Min)
	Mean     float64 `json:"mean"`     // Arithmetisches Mittel
	Median   float64 `json:"median"`   // 50. Perzentil
	Variance float64 `json:"variance"` // Varianz
	StdDev   float64 `json:"std_dev"`  // Standardabweichung
	Q1       float64 `json:"q1"`       // 25. Perzentil (Unteres Quartil)
	Q3       float64 `json:"q3"`       // 75. Perzentil (Oberes Quartil)
	IQR      float64 `json:"iqr"`      // Interquartilsabstand (Q3 - Q1)
}

// CalcMin ermittelt den kleinsten Wert im Datensatz
func CalcMin(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	return floats.Min(data)
}

// CalcMax ermittelt den größten Wert im Datensatz
func CalcMax(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	return floats.Max(data)
}

// CalcRange berechnet die absolute Spannweite (Differenz zwischen Max und Min)
func CalcRange(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	return floats.Max(data) - floats.Min(data)
}

// CalcMean berechnet das arithmetische Mittel
func CalcMean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	return stat.Mean(data, nil)
}

// WeightedMean berechnet den gewichteten Mittelwert: Summe(values[i] * weights[i]) / Summe(weights[i])
func CalcWeightedMean(values, weights []float64) (float64, error) {
	var (
		ErrLengthMismatch = errors.New("input slices must have the same length")
		ErrEmptyInput     = errors.New("input slices cannot be empty")
		ErrZeroWeightSum  = errors.New("sum of weights cannot be zero")
	)
	if len(values) == 0 || len(weights) == 0 {
		return 0, ErrEmptyInput
	}
	if len(values) != len(weights) {
		return 0, ErrLengthMismatch
	}
	if floats.Sum(weights) == 0 {
		return 0, ErrZeroWeightSum
	}
	return stat.Mean(values, weights), nil
}

// CalcMedian berechnet den Zentralwert (Median)
func CalcMedian(data []float64) float64 {
	return CalcPercentile(data, 50.0)
}

// CalcVariance berechnet die Varianz (Nutzt das zuvor berechnete Mean)
func CalcVariance(data []float64, mean float64) float64 {
	_ = mean // Signatur bleibt kompatibel
	n := len(data)
	if n < 2 {
		return 0
	}
	// stat.Variance = Stichprobenvarianz (n-1), hier auf Populationsvarianz (n) umrechnen
	return stat.Variance(data, nil) * float64(n-1) / float64(n)
}

func CalcSampleVariance(data []float64, mean float64) float64 {
	_ = mean // Signatur bleibt kompatibel
	if len(data) < 2 {
		return 0
	}
	return stat.Variance(data, nil)
}

func CalcCovariance(xData, yData []float64, meanX, meanY float64) float64 {
	_ = meanX // Signatur bleibt kompatibel
	_ = meanY
	n := len(xData)
	if n < 2 || n != len(yData) {
		return 0
	}
	// stat.Covariance = Stichprobenkovarianz (n-1), hier auf Populationskovarianz (n) umrechnen
	return stat.Covariance(xData, yData, nil) * float64(n-1) / float64(n)
}

func CalcStdDev(variance float64) float64 {
	return math.Sqrt(variance)
}

// CalcPercentile berechnet ein beliebiges Perzentil (p liegt zwischen 0.0 und 100.0)
func CalcPercentile(data []float64, p float64) float64 {
	n := len(data)
	if n == 0 {
		return 0
	}
	if p <= 0 {
		return floats.Min(data)
	}
	if p >= 100 {
		return floats.Max(data)
	}

	sorted := make([]float64, n)
	copy(sorted, data)
	sort.Float64s(sorted)

	return stat.Quantile(p/100.0, stat.LinInterp, sorted, nil)
}

// CalcIQR berechnet den Interquartilsabstand (Interquartile Range)
func CalcIQR(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	q1 := CalcPercentile(data, 25.0)
	q3 := CalcPercentile(data, 75.0)
	return q3 - q1
}

// GetSummary berechnet alle Kennwerte und gibt ein fertiges Struct zurück
func GetSummary(data []float64) (SummaryStats, error) {
	if len(data) == 0 {
		return SummaryStats{}, fmt.Errorf("datensatz ist leer")
	}

	mean := CalcMean(data)
	variance := CalcVariance(data, mean)
	q1 := CalcPercentile(data, 25.0)
	q3 := CalcPercentile(data, 75.0)

	return SummaryStats{
		Min:      CalcMin(data),
		Max:      CalcMax(data),
		Range:    CalcRange(data),
		Mean:     mean,
		Median:   CalcPercentile(data, 50.0),
		Variance: variance,
		StdDev:   CalcStdDev(variance),
		Q1:       q1,
		Q3:       q3,
		IQR:      q3 - q1,
	}, nil
}

func CalcMode(data []float64) []float64 {
	if len(data) == 0 {
		return nil
	}

	frequencies := make(map[float64]int)
	maxCount := 0

	for _, v := range data {
		frequencies[v]++
		if frequencies[v] > maxCount {
			maxCount = frequencies[v]
		}
	}

	var modes []float64
	for value, count := range frequencies {
		if count == maxCount {
			modes = append(modes, value)
		}
	}
	return modes
}

func CalcCoefficientOfVariation(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	mean := stat.Mean(data, nil)
	if mean == 0 {
		return 0
	}
	variance := CalcVariance(data, mean)
	return math.Sqrt(variance) / mean
}

func CalcSum(values []float64) float64 {
	return floats.Sum(values)
}
