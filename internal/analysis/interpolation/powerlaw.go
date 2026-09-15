package interpolation

import (
	"math"
	"sort"
	"time"

	"gonum.org/v1/gonum/stat"

	"wind_analysis/internal/database"
	"wind_analysis/models"
)

// HellmannExponentResult speichert das Ergebnis der Hellmann-Exponent Berechnung
type HellmannExponentResult struct {
	Time     time.Time
	Location models.Location
	Alpha    float64
}

// DescriptiveStats speichert kompakte deskriptive Kennzahlen einer Stichprobe.
type DescriptiveStats struct {
	//Lageparameter
	Min    float64
	Max    float64
	Mean   float64
	Median float64
	// Streuungsparameter
	StandardDeviation      float64
	Variance               float64
	CoefficientOfVariation float64
	// Formparameter
	Skewness float64
	Kurtosis float64 // "klassische" Kurtosis (Exzess + 3)
	//Quantile
	Quantiles          map[float64]float64 // z.B. 0.25, 0.5, 0.75
	InnerquantileRange float64
	Percentil          map[float64]float64 // z.B. 10, 90
	// Anzahl der Stichproben
	SampleCount int
}

// ValidationResult speichert die Ergebnisse der Modellvalidierung
type ValidationResult struct {
	Height      float64
	MAE         float64 // Mean Absolute Error
	RMSE        float64 // Root Mean Square Error
	Correlation float64 // Pearson correlation coefficient
	Bias        float64 // Mean Bias Error
	SampleCount int

	PredictedSummary DescriptiveStats
	ActualSummary    DescriptiveStats
	ErrorSummary     DescriptiveStats
}

// ValidationStats speichert detaillierte Statistiken für die Validierung
type ValidationStats struct {
	Height              float64
	PredictedValues     []float64
	ActualValues        []float64
	MeanAbsoluteError   float64
	RootMeanSquareError float64
	Correlation         float64
	SampleCount         int

	PredictedSummary DescriptiveStats
	ActualSummary    DescriptiveStats
	ErrorSummary     DescriptiveStats
}

const (
	unknownLocationName = "Unbekannt"
	referenceHeight10m  = 10.0
)

var defaultValidationHeights = []float64{80, 120, 180, 200}

// CalculateHellmannExponent berechnet den Hellmann-Exponent (alpha) basierend auf zwei bekannten Windgeschwindigkeiten und deren Höhen.
func CalculateHellmannExponent(v1, v2 float64, h1, h2 float64) float64 {
	if v1 <= 0 || v2 <= 0 || h1 <= 0 || h2 <= 0 {
		return math.NaN() // Ungültige Eingaben
	}
	return math.Log(v2/v1) / math.Log(h2/h1)
}

// InterpolateWindSpeed berechnet die Windgeschwindigkeit in einer Zielhöhe basierend auf einer bekannten Windgeschwindigkeit und dem Hellmann-Exponent.
func InterpolateWindSpeed(v_ref, h_ref, h_target, alpha float64) float64 {
	if v_ref <= 0 || h_ref <= 0 || h_target <= 0 {
		return math.NaN() // Ungültige Eingaben
	}
	return v_ref * math.Pow(h_target/h_ref, alpha)
}

// CalculateHellmannExponentsForDataset berechnet den Hellmann-Exponent für alle gültigen Datensätze
// basierend auf den Windgeschwindigkeiten in 10m und 100m Höhe.
func CalculateHellmannExponentsForDataset(records []models.WindRecord) []HellmannExponentResult {
	var results []HellmannExponentResult

	for _, record := range records {
		v10m := database.SafeVal(record.WindData.WindSpeed_10m)
		v100m := database.SafeVal(record.WindData.WindSpeed_100m)

		// Nur berechnen, wenn beide Werte gültig und positiv sind
		if v10m > 0 && v100m > 0 {
			alpha := CalculateHellmannExponent(v10m, v100m, 10, 100)

			// Prüfen, ob Alpha ein gültiger Wert ist (nicht NaN und im plausiblen Bereich)
			if !math.IsNaN(alpha) && alpha > 0 && alpha < 1.0 {
				results = append(results, HellmannExponentResult{
					Time:     record.Time,
					Location: record.Location,
					Alpha:    alpha,
				})
			}
		}
	}

	return results
}

func buildExponentKey(t time.Time, locationName string) string {
	if locationName == "" {
		locationName = unknownLocationName
	}
	return t.Format(time.RFC3339) + "_" + locationName
}

func buildExponentMap(exponents []HellmannExponentResult) map[string]float64 {
	exponentMap := make(map[string]float64, len(exponents))
	for _, exp := range exponents {
		exponentMap[buildExponentKey(exp.Time, exp.Location.Name)] = exp.Alpha
	}
	return exponentMap
}

func emptyDescriptiveStats() DescriptiveStats {
	return DescriptiveStats{
		Mean:                   math.NaN(),
		Median:                 math.NaN(),
		StandardDeviation:      math.NaN(),
		Variance:               math.NaN(),
		CoefficientOfVariation: math.NaN(),
		Skewness:               math.NaN(),
		Kurtosis:               math.NaN(),
	}
}

func newValidationStats(height float64) ValidationStats {
	return ValidationStats{
		Height:              height,
		PredictedValues:     []float64{},
		ActualValues:        []float64{},
		MeanAbsoluteError:   math.NaN(),
		RootMeanSquareError: math.NaN(),
		Correlation:         math.NaN(),
		PredictedSummary:    emptyDescriptiveStats(),
		ActualSummary:       emptyDescriptiveStats(),
		ErrorSummary:        emptyDescriptiveStats(),
	}
}

func initValidationStatsByHeight(heights []float64) map[float64]ValidationStats {
	statsByHeight := make(map[float64]ValidationStats, len(heights))
	for _, height := range heights {
		statsByHeight[height] = newValidationStats(height)
	}
	return statsByHeight
}

func buildErrors(predicted, actual []float64) []float64 {
	if len(predicted) != len(actual) {
		return nil
	}
	errs := make([]float64, len(predicted))
	for i := range predicted {
		errs[i] = predicted[i] - actual[i]
	}
	return errs
}

func calculateDescriptiveStats(values []float64) DescriptiveStats {
	if len(values) == 0 {
		return emptyDescriptiveStats()
	}

	mean := stat.Mean(values, nil)
	variance := stat.Variance(values, nil)
	stdDev := math.Sqrt(variance)

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	median := stat.Quantile(0.5, stat.Empirical, sorted, nil)

	cv := math.NaN()
	if mean != 0 && !math.IsNaN(mean) {
		cv = stdDev / mean
	}

	skew := stat.Skew(values, nil)
	// Gonum liefert Exzess-Kurtosis; hier in "klassische" Kurtosis umgerechnet.
	kurtosis := stat.ExKurtosis(values, nil) + 3.0

	return DescriptiveStats{
		Mean:                   mean,
		Median:                 median,
		StandardDeviation:      stdDev,
		Variance:               variance,
		CoefficientOfVariation: cv,
		Skewness:               skew,
		Kurtosis:               kurtosis,
		SampleCount:            len(values),
	}
}

func finalizeValidationStats(heightMap map[float64]ValidationStats) {
	for height, stats := range heightMap {
		if stats.SampleCount == 0 {
			continue
		}
		stats.MeanAbsoluteError = calculateMAE(stats.PredictedValues, stats.ActualValues)
		stats.RootMeanSquareError = calculateRMSE(stats.PredictedValues, stats.ActualValues)
		stats.Correlation = calculateCorrelation(stats.PredictedValues, stats.ActualValues)

		stats.PredictedSummary = calculateDescriptiveStats(stats.PredictedValues)
		stats.ActualSummary = calculateDescriptiveStats(stats.ActualValues)
		stats.ErrorSummary = calculateDescriptiveStats(buildErrors(stats.PredictedValues, stats.ActualValues))

		heightMap[height] = stats
	}
}

// ValidatePowerLawModelDetailed validiert das Power-Law-Modell und liefert detaillierte Stats je Höhe.
func ValidatePowerLawModelDetailed(records []models.WindRecord, exponents []HellmannExponentResult) map[float64]ValidationStats {
	exponentMap := buildExponentMap(exponents)
	validationData := initValidationStatsByHeight(defaultValidationHeights)

	for _, record := range records {
		key := buildExponentKey(record.Time, record.Location.Name)
		alpha, exists := exponentMap[key]
		if !exists || math.IsNaN(alpha) {
			continue
		}

		v10m := database.SafeVal(record.WindData.WindSpeed_10m)
		if v10m <= 0 {
			continue
		}

		for _, height := range defaultValidationHeights {
			vActual := database.SafeVal(getWindSpeedAtHeight(record.WindData, height))
			if vActual <= 0 {
				continue
			}

			vPredicted := InterpolateWindSpeed(v10m, referenceHeight10m, height, alpha)
			if math.IsNaN(vPredicted) {
				continue
			}

			stats := validationData[height]
			stats.PredictedValues = append(stats.PredictedValues, vPredicted)
			stats.ActualValues = append(stats.ActualValues, vActual)
			stats.SampleCount++
			validationData[height] = stats
		}
	}

	finalizeValidationStats(validationData)
	return validationData
}

// ValidatePowerLawModelByLocationDetailed validiert das Power-Law-Modell je Ort und Höhe.
func ValidatePowerLawModelByLocationDetailed(records []models.WindRecord, exponents []HellmannExponentResult) map[string]map[float64]ValidationStats {
	exponentMap := buildExponentMap(exponents)
	byLocation := make(map[string]map[float64]ValidationStats)

	for _, record := range records {
		loc := record.Location.Name
		if loc == "" {
			loc = unknownLocationName
		}

		key := buildExponentKey(record.Time, loc)
		alpha, exists := exponentMap[key]
		if !exists || math.IsNaN(alpha) {
			continue
		}

		v10m := database.SafeVal(record.WindData.WindSpeed_10m)
		if v10m <= 0 {
			continue
		}

		if _, ok := byLocation[loc]; !ok {
			byLocation[loc] = initValidationStatsByHeight(defaultValidationHeights)
		}

		for _, height := range defaultValidationHeights {
			vActual := database.SafeVal(getWindSpeedAtHeight(record.WindData, height))
			if vActual <= 0 {
				continue
			}

			vPredicted := InterpolateWindSpeed(v10m, referenceHeight10m, height, alpha)
			if math.IsNaN(vPredicted) {
				continue
			}

			stats := byLocation[loc][height]
			stats.PredictedValues = append(stats.PredictedValues, vPredicted)
			stats.ActualValues = append(stats.ActualValues, vActual)
			stats.SampleCount++
			byLocation[loc][height] = stats
		}
	}

	for loc := range byLocation {
		finalizeValidationStats(byLocation[loc])
	}
	return byLocation
}

func toValidationResults(detailed map[float64]ValidationStats) map[float64]ValidationResult {
	results := make(map[float64]ValidationResult, len(detailed))
	for height, stats := range detailed {
		if stats.SampleCount == 0 {
			continue
		}
		results[height] = ValidationResult{
			Height:           height,
			MAE:              stats.MeanAbsoluteError,
			RMSE:             stats.RootMeanSquareError,
			Correlation:      stats.Correlation,
			SampleCount:      stats.SampleCount,
			PredictedSummary: stats.PredictedSummary,
			ActualSummary:    stats.ActualSummary,
			ErrorSummary:     stats.ErrorSummary,
		}
	}
	return results
}

// ValidatePowerLawModel validiert das Power-Law-Modell mit zusätzlichen Höhen
// vergleicht die interpolierten Werte mit den tatsächlichen Messwerten.
func ValidatePowerLawModel(records []models.WindRecord, exponents []HellmannExponentResult) map[float64]ValidationResult {
	return toValidationResults(ValidatePowerLawModelDetailed(records, exponents))
}

// ValidatePowerLawModelByLocation liefert MAE/RMSE/Korrelation je Ort und Höhe.
func ValidatePowerLawModelByLocation(records []models.WindRecord, exponents []HellmannExponentResult) map[string]map[float64]ValidationResult {
	detailed := ValidatePowerLawModelByLocationDetailed(records, exponents)
	results := make(map[string]map[float64]ValidationResult, len(detailed))

	for loc, heightMap := range detailed {
		results[loc] = toValidationResults(heightMap)
	}
	return results
}

func filterRecords(records []models.WindRecord, keep func(models.WindRecord) bool) []models.WindRecord {
	filtered := make([]models.WindRecord, 0, len(records))
	for _, record := range records {
		if keep(record) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

// FilterRecordsByYear filtert WindRecords nach einem bestimmten Jahr
func FilterRecordsByYear(records []models.WindRecord, year int) []models.WindRecord {
	return filterRecords(records, func(record models.WindRecord) bool {
		return record.Time.Year() == year
	})
}

// FilterRecordsFromYear filtert WindRecords ab einem bestimmten Jahr (inklusive)
func FilterRecordsFromYear(records []models.WindRecord, year int) []models.WindRecord {
	return filterRecords(records, func(record models.WindRecord) bool {
		return record.Time.Year() >= year
	})
}

// getWindSpeedAtHeight gibt die Windgeschwindigkeit für eine bestimmte Höhe zurück
func getWindSpeedAtHeight(windData models.WindData, height float64) *float64 {
	switch height {
	case 10:
		return windData.WindSpeed_10m
	case 80:
		return windData.WindSpeed_80m
	case 100:
		return windData.WindSpeed_100m
	case 120:
		return windData.WindSpeed_120m
	case 180:
		return windData.WindSpeed_180m
	case 200:
		return windData.WindSpeed_200m
	default:
		return nil // Höhe nicht verfügbar
	}
}

func calculateMAE(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}

	errs := make([]float64, len(predicted))
	for i := range predicted {
		errs[i] = math.Abs(predicted[i] - actual[i])
	}
	return stat.Mean(errs, nil)
}

func calculateRMSE(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}

	sq := make([]float64, len(predicted))
	for i := range predicted {
		diff := predicted[i] - actual[i]
		sq[i] = diff * diff
	}
	return math.Sqrt(stat.Mean(sq, nil))
}

func calculateCorrelation(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}
	return stat.Correlation(predicted, actual, nil)
}
