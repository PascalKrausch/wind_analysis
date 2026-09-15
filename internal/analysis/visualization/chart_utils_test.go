package visualization

import (
	"testing"
	"time"
)

func TestNewTimeSeriesConverter(t *testing.T) {
	converter := NewTimeSeriesConverter("2006-01-02")

	if converter.TimeFormat != "2006-01-02" {
		t.Errorf("Expected format '2006-01-02', got '%s'", converter.TimeFormat)
	}

	// Test default format
	defaultConverter := NewTimeSeriesConverter("")
	if defaultConverter.TimeFormat != "2006-01-02 15:04" {
		t.Errorf("Expected default format '2006-01-02 15:04', got '%s'", defaultConverter.TimeFormat)
	}
}

func TestConvertTimesToXAxis(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 2, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 3, 12, 0, 0, 0, time.UTC),
	}

	converter := NewTimeSeriesConverter("2006-01-02")
	result := converter.ConvertTimesToXAxis(times)

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	if result[0] != "2022-01-01" {
		t.Errorf("Expected '2022-01-01', got '%s'", result[0])
	}
}

func TestConvertTimesToXAxisDaily(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 2, 15, 30, 0, 0, time.UTC),
	}

	result := ConvertTimesToXAxisDaily(times)

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0] != "2022-01-01" {
		t.Errorf("Expected '2022-01-01', got '%s'", result[0])
	}
}

func TestConvertTimesToXAxisHourly(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 1, 15, 30, 0, 0, time.UTC),
	}

	result := ConvertTimesToXAxisHourly(times)

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0] != "2022-01-01 12:00" {
		t.Errorf("Expected '2022-01-01 12:00', got '%s'", result[0])
	}
}

func TestConvertTimesToXAxisMonthly(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 2, 1, 12, 0, 0, 0, time.UTC),
	}

	result := ConvertTimesToXAxisMonthly(times)

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0] != "2022-01" {
		t.Errorf("Expected '2022-01', got '%s'", result[0])
	}
}

func TestFloat64ToStringSlice(t *testing.T) {
	data := []float64{1.234, 2.567, 3.891}

	result := Float64ToStringSlice(data)

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	if result[0] != "1.23" {
		t.Errorf("Expected '1.23', got '%s'", result[0])
	}
}

func TestIntToStringSlice(t *testing.T) {
	data := []int{1, 2, 3}

	result := IntToStringSlice(data)

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	if result[0] != "1" {
		t.Errorf("Expected '1', got '%s'", result[0])
	}
}

func TestGroupByMonth(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 2, 1, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{1.0, 2.0, 3.0}

	monthlyValues, _ := GroupByMonth(times, values)

	if len(monthlyValues) != 2 {
		t.Errorf("Expected 2 months, got %d", len(monthlyValues))
	}

	if len(monthlyValues["2022-01"]) != 2 {
		t.Errorf("Expected 2 values for January, got %d", len(monthlyValues["2022-01"]))
	}

	if len(monthlyValues["2022-02"]) != 1 {
		t.Errorf("Expected 1 value for February, got %d", len(monthlyValues["2022-02"]))
	}
}

func TestGroupByMonthMismatchedLengths(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{1.0, 2.0}

	monthlyValues, monthlyTimes := GroupByMonth(times, values)

	if monthlyValues != nil || monthlyTimes != nil {
		t.Error("Expected nil for mismatched lengths")
	}
}

func TestGroupByYear(t *testing.T) {
	times := []time.Time{
		time.Date(2021, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2021, 6, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{1.0, 2.0, 3.0}

	yearlyValues, _ := GroupByYear(times, values)

	if len(yearlyValues) != 2 {
		t.Errorf("Expected 2 years, got %d", len(yearlyValues))
	}

	if len(yearlyValues["2021"]) != 2 {
		t.Errorf("Expected 2 values for 2021, got %d", len(yearlyValues["2021"]))
	}

	if len(yearlyValues["2022"]) != 1 {
		t.Errorf("Expected 1 value for 2022, got %d", len(yearlyValues["2022"]))
	}
}

func TestCalculateMonthlyAverage(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 2, 1, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{1.0, 3.0, 5.0}

	averages, counts := CalculateMonthlyAverage(times, values)

	if len(averages) != 2 {
		t.Errorf("Expected 2 months, got %d", len(averages))
	}

	// January average should be (1.0 + 3.0) / 2 = 2.0
	if averages["2022-01"] != 2.0 {
		t.Errorf("Expected January average 2.0, got %f", averages["2022-01"])
	}

	if counts["2022-01"] != 2 {
		t.Errorf("Expected January count 2, got %d", counts["2022-01"])
	}

	// February average should be 5.0
	if averages["2022-02"] != 5.0 {
		t.Errorf("Expected February average 5.0, got %f", averages["2022-02"])
	}
}

func TestCalculateYearlyAverage(t *testing.T) {
	times := []time.Time{
		time.Date(2021, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2021, 6, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{1.0, 3.0, 5.0}

	averages, counts := CalculateYearlyAverage(times, values)

	if len(averages) != 2 {
		t.Errorf("Expected 2 years, got %d", len(averages))
	}

	// 2021 average should be (1.0 + 3.0) / 2 = 2.0
	if averages["2021"] != 2.0 {
		t.Errorf("Expected 2021 average 2.0, got %f", averages["2021"])
	}

	if counts["2021"] != 2 {
		t.Errorf("Expected 2021 count 2, got %d", counts["2021"])
	}

	// 2022 average should be 5.0
	if averages["2022"] != 5.0 {
		t.Errorf("Expected 2022 average 5.0, got %f", averages["2022"])
	}
}

func TestFilterByTimeRange(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 2, 1, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{1.0, 2.0, 3.0}

	start := time.Date(2022, 1, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2022, 1, 20, 0, 0, 0, 0, time.UTC)

	filteredTimes, filteredValues := FilterByTimeRange(times, values, start, end)

	if len(filteredTimes) != 1 {
		t.Errorf("Expected 1 filtered item, got %d", len(filteredTimes))
	}

	if filteredValues[0] != 2.0 {
		t.Errorf("Expected filtered value 2.0, got %f", filteredValues[0])
	}
}

func TestFilterByHour(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 1, 15, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 1, 12, 30, 0, 0, time.UTC),
	}
	values := []float64{1.0, 2.0, 3.0}

	filteredTimes, filteredValues := FilterByHour(times, values, 12)

	if len(filteredTimes) != 2 {
		t.Errorf("Expected 2 filtered items, got %d", len(filteredTimes))
	}

	if filteredValues[0] != 1.0 {
		t.Errorf("Expected first filtered value 1.0, got %f", filteredValues[0])
	}
}

func TestFilterBySeason(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),  // Winter
		time.Date(2022, 4, 1, 12, 0, 0, 0, time.UTC),  // Spring
		time.Date(2022, 7, 1, 12, 0, 0, 0, time.UTC),  // Summer
	}
	values := []float64{1.0, 2.0, 3.0}

	filteredTimes, filteredValues := FilterBySeason(times, values, "winter")

	if len(filteredTimes) != 1 {
		t.Errorf("Expected 1 filtered item for winter, got %d", len(filteredTimes))
	}

	if filteredValues[0] != 1.0 {
		t.Errorf("Expected filtered value 1.0, got %f", filteredValues[0])
	}
}

func TestFilterBySeasonInvalid(t *testing.T) {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{1.0}

	filteredTimes, filteredValues := FilterBySeason(times, values, "invalid")

	if filteredTimes != nil || filteredValues != nil {
		t.Error("Expected nil for invalid season")
	}
}

func TestCalculateStatistics(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	min, max, mean, stdDev := CalculateStatistics(values)

	if min != 1.0 {
		t.Errorf("Expected min 1.0, got %f", min)
	}

	if max != 5.0 {
		t.Errorf("Expected max 5.0, got %f", max)
	}

	if mean != 3.0 {
		t.Errorf("Expected mean 3.0, got %f", mean)
	}

	// Standard deviation should be sqrt(2) ≈ 1.414
	expectedStdDev := 1.4142135623730951
	if stdDev < expectedStdDev-0.01 || stdDev > expectedStdDev+0.01 {
		t.Errorf("Expected stdDev approx %.2f, got %f", expectedStdDev, stdDev)
	}
}

func TestCalculateStatisticsEmpty(t *testing.T) {
	values := []float64{}

	min, max, mean, stdDev := CalculateStatistics(values)

	if min != 0 || max != 0 || mean != 0 || stdDev != 0 {
		t.Error("Expected all zeros for empty input")
	}
}

func TestNormalizeData(t *testing.T) {
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	normalized := NormalizeData(values)

	if len(normalized) != 5 {
		t.Errorf("Expected 5 items, got %d", len(normalized))
	}

	// First value should be 0.0 (min)
	if normalized[0] != 0.0 {
		t.Errorf("Expected first value 0.0, got %f", normalized[0])
	}

	// Last value should be 1.0 (max)
	if normalized[4] != 1.0 {
		t.Errorf("Expected last value 1.0, got %f", normalized[4])
	}
}

func TestNormalizeDataEmpty(t *testing.T) {
	values := []float64{}

	normalized := NormalizeData(values)

	if len(normalized) != 0 {
		t.Errorf("Expected empty result, got %d items", len(normalized))
	}
}

func TestNormalizeDataConstant(t *testing.T) {
	values := []float64{5.0, 5.0, 5.0}

	normalized := NormalizeData(values)

	// Should return original values when all are the same
	if len(normalized) != 3 {
		t.Errorf("Expected 3 items, got %d", len(normalized))
	}

	for i, v := range normalized {
		if v != 5.0 {
			t.Errorf("Expected value 5.0 at index %d, got %f", i, v)
		}
	}
}

func TestCreateColorScale(t *testing.T) {
	// Test low value (should be blue)
	colorLow := CreateColorScale(0.0, 0.0, 10.0)
	if colorLow == "" {
		t.Error("Expected non-empty color string for low value")
	}

	// Test high value (should be red)
	colorHigh := CreateColorScale(10.0, 0.0, 10.0)
	if colorHigh == "" {
		t.Error("Expected non-empty color string for high value")
	}

	// Test middle value (should be yellow/green)
	colorMid := CreateColorScale(5.0, 0.0, 10.0)
	if colorMid == "" {
		t.Error("Expected non-empty color string for middle value")
	}
}

func TestCreateColorScaleConstantRange(t *testing.T) {
	color := CreateColorScale(5.0, 5.0, 5.0)

	if color != "#50a3ba" {
		t.Errorf("Expected default color '#50a3ba', got '%s'", color)
	}
}

func TestCreateColorScaleBelowMin(t *testing.T) {
	color := CreateColorScale(-1.0, 0.0, 10.0)

	// Should handle values below min gracefully
	if color == "" {
		t.Error("Expected non-empty color string for value below min")
	}
}

func TestCreateColorScaleAboveMax(t *testing.T) {
	color := CreateColorScale(15.0, 0.0, 10.0)

	// Should handle values above max gracefully
	if color == "" {
		t.Error("Expected non-empty color string for value above max")
	}
}
