package visualization

import (
	"fmt"
	"time"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

// TimeSeriesConverter konvertiert Zeitreihen für Chart-Darstellung
type TimeSeriesConverter struct {
	TimeFormat string
}

// NewTimeSeriesConverter erstellt einen neuen TimeSeriesConverter
func NewTimeSeriesConverter(format string) *TimeSeriesConverter {
	if format == "" {
		format = "2006-01-02 15:04" // Standardformat
	}
	return &TimeSeriesConverter{TimeFormat: format}
}

// ConvertTimesToXAxis konvertiert eine Zeitreihe in X-Achsen-Strings
func (tsc *TimeSeriesConverter) ConvertTimesToXAxis(times []time.Time) []string {
	xAxis := make([]string, len(times))
	for i, t := range times {
		xAxis[i] = t.Format(tsc.TimeFormat)
	}
	return xAxis
}

// ConvertTimesToXAxisDaily konvertiert Zeiten in tägliche X-Achsen-Strings
func ConvertTimesToXAxisDaily(times []time.Time) []string {
	converter := NewTimeSeriesConverter("2006-01-02")
	return converter.ConvertTimesToXAxis(times)
}

// ConvertTimesToXAxisHourly konvertiert Zeiten in stündliche X-Achsen-Strings
func ConvertTimesToXAxisHourly(times []time.Time) []string {
	converter := NewTimeSeriesConverter("2006-01-02 15:00")
	return converter.ConvertTimesToXAxis(times)
}

// ConvertTimesToXAxisMonthly konvertiert Zeiten in monatliche X-Achsen-Strings
func ConvertTimesToXAxisMonthly(times []time.Time) []string {
	converter := NewTimeSeriesConverter("2006-01")
	return converter.ConvertTimesToXAxis(times)
}

// Float64ToStringSlice konvertiert ein Float64-Slice in ein String-Slice
func Float64ToStringSlice(data []float64) []string {
	result := make([]string, len(data))
	for i, d := range data {
		result[i] = fmt.Sprintf("%.2f", d)
	}
	return result
}

// IntToStringSlice konvertiert ein Int-Slice in ein String-Slice
func IntToStringSlice(data []int) []string {
	result := make([]string, len(data))
	for i, d := range data {
		result[i] = fmt.Sprintf("%d", d)
	}
	return result
}

// GroupByMonth gruppiert Daten nach Monaten
func GroupByMonth(times []time.Time, values []float64) (map[string][]float64, map[string][]time.Time) {
	if len(times) != len(values) {
		return nil, nil
	}

	monthlyValues := make(map[string][]float64)
	monthlyTimes := make(map[string][]time.Time)

	for i, t := range times {
		key := t.Format("2006-01")
		monthlyValues[key] = append(monthlyValues[key], values[i])
		monthlyTimes[key] = append(monthlyTimes[key], t)
	}

	return monthlyValues, monthlyTimes
}

// GroupByYear gruppiert Daten nach Jahren
func GroupByYear(times []time.Time, values []float64) (map[string][]float64, map[string][]time.Time) {
	if len(times) != len(values) {
		return nil, nil
	}

	yearlyValues := make(map[string][]float64)
	yearlyTimes := make(map[string][]time.Time)

	for i, t := range times {
		key := fmt.Sprintf("%d", t.Year())
		yearlyValues[key] = append(yearlyValues[key], values[i])
		yearlyTimes[key] = append(yearlyTimes[key], t)
	}

	return yearlyValues, yearlyTimes
}

// CalculateMonthlyAverage berechnet monatliche Durchschnittswerte
func CalculateMonthlyAverage(times []time.Time, values []float64) (map[string]float64, map[string]int) {
	monthlyValues, _ := GroupByMonth(times, values)

	averages := make(map[string]float64)
	counts := make(map[string]int)

	for month, vals := range monthlyValues {
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		averages[month] = sum / float64(len(vals))
		counts[month] = len(vals)
	}

	return averages, counts
}

// CalculateYearlyAverage berechnet jährliche Durchschnittswerte
func CalculateYearlyAverage(times []time.Time, values []float64) (map[string]float64, map[string]int) {
	yearlyValues, _ := GroupByYear(times, values)

	averages := make(map[string]float64)
	counts := make(map[string]int)

	for year, vals := range yearlyValues {
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		averages[year] = sum / float64(len(vals))
		counts[year] = len(vals)
	}

	return averages, counts
}

// FilterByTimeRange filtert Daten nach einem Zeitbereich
func FilterByTimeRange(times []time.Time, values []float64, start, end time.Time) ([]time.Time, []float64) {
	if len(times) != len(values) {
		return nil, nil
	}

	var filteredTimes []time.Time
	var filteredValues []float64

	for i, t := range times {
		if (t.Equal(start) || t.After(start)) && (t.Equal(end) || t.Before(end)) {
			filteredTimes = append(filteredTimes, t)
			filteredValues = append(filteredValues, values[i])
		}
	}

	return filteredTimes, filteredValues
}

// FilterByHour filtert Daten nach einer bestimmten Stunde
func FilterByHour(times []time.Time, values []float64, hour int) ([]time.Time, []float64) {
	if len(times) != len(values) {
		return nil, nil
	}

	var filteredTimes []time.Time
	var filteredValues []float64

	for i, t := range times {
		if t.Hour() == hour {
			filteredTimes = append(filteredTimes, t)
			filteredValues = append(filteredValues, values[i])
		}
	}

	return filteredTimes, filteredValues
}

// FilterBySeason filtert Daten nach Saison (Winter: 12-2, Frühling: 3-5, Sommer: 6-8, Herbst: 9-11)
func FilterBySeason(times []time.Time, values []float64, season string) ([]time.Time, []float64) {
	if len(times) != len(values) {
		return nil, nil
	}

	var months []int
	switch season {
	case "winter":
		months = []int{12, 1, 2}
	case "spring":
		months = []int{3, 4, 5}
	case "summer":
		months = []int{6, 7, 8}
	case "autumn":
		months = []int{9, 10, 11}
	default:
		return nil, nil
	}

	var filteredTimes []time.Time
	var filteredValues []float64

	for i, t := range times {
		for _, m := range months {
			if int(t.Month()) == m {
				filteredTimes = append(filteredTimes, t)
				filteredValues = append(filteredValues, values[i])
				break
			}
		}
	}

	return filteredTimes, filteredValues
}

// CalculateStatistics berechnet grundlegende Statistiken für ein Dataset
func CalculateStatistics(values []float64) (min, max, mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}

	min = floats.Min(values)
	max = floats.Max(values)
	mean = stat.Mean(values, nil)
	stdDev = stat.PopStdDev(values, nil)

	return min, max, mean, stdDev
}

// NormalizeData normalisiert Daten auf einen Bereich [0, 1]
func NormalizeData(values []float64) []float64 {
	if len(values) == 0 {
		return values
	}

	min, max, _, _ := CalculateStatistics(values)
	if max == min {
		return values
	}

	normalized := make([]float64, len(values))
	for i, v := range values {
		normalized[i] = (v - min) / (max - min)
	}

	return normalized
}

// CreateColorScale erstellt eine Farbskala basierend auf einem Wert
func CreateColorScale(value, min, max float64) string {
	if max == min {
		return "#50a3ba" // Standardfarbe
	}

	normalized := (value - min) / (max - min)
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}

	// Farbskala von Blau (niedrig) zu Gelb (mittel) zu Rot (hoch)
	if normalized < 0.5 {
		// Blau zu Gelb
		r := int(normalized * 2 * 255)
		g := int(normalized * 2 * 255)
		b := 255 - int(normalized*2*100)
		return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	} else {
		// Gelb zu Rot
		r := 255
		g := 255 - int((normalized-0.5)*2*255)
		b := 155 - int((normalized-0.5)*2*155)
		return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	}
}
