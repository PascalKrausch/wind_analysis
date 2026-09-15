package interpolation

import (
	"math"
	"testing"
	"time"

	"wind_analysis/models"
)

func TestCalculateHellmannExponent(t *testing.T) {
	tests := []struct {
		name string
		v1, v2, h1, h2 float64
		expected float64
	}{
		{
			name: "Normal case",
			v1: 3.0, v2: 4.0, h1: 10, h2: 100,
			expected: math.Log(4.0/3.0) / math.Log(100.0/10.0),
		},
		{
			name: "Zero velocity",
			v1: 0, v2: 4.0, h1: 10, h2: 100,
			expected: math.NaN(),
		},
		{
			name: "Negative velocity",
			v1: -1.0, v2: 4.0, h1: 10, h2: 100,
			expected: math.NaN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateHellmannExponent(tt.v1, tt.v2, tt.h1, tt.h2)
			if math.IsNaN(tt.expected) {
				if !math.IsNaN(result) {
					t.Errorf("Expected NaN, got %v", result)
				}
			} else {
				if math.Abs(result-tt.expected) > 1e-10 {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestInterpolateWindSpeed(t *testing.T) {
	tests := []struct {
		name string
		v_ref, h_ref, h_target, alpha float64
		expected float64
	}{
		{
			name: "Normal case",
			v_ref: 3.0, h_ref: 10, h_target: 100, alpha: 0.15,
			expected: 3.0 * math.Pow(100.0/10.0, 0.15),
		},
		{
			name: "Zero reference velocity",
			v_ref: 0, h_ref: 10, h_target: 100, alpha: 0.15,
			expected: math.NaN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InterpolateWindSpeed(tt.v_ref, tt.h_ref, tt.h_target, tt.alpha)
			if math.IsNaN(tt.expected) {
				if !math.IsNaN(result) {
					t.Errorf("Expected NaN, got %v", result)
				}
			} else {
				if math.Abs(result-tt.expected) > 1e-10 {
					t.Errorf("Expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestCalculateHellmannExponentsForDataset(t *testing.T) {
	records := []models.WindRecord{
		{
			Time: time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
			Location: models.Location{Name: "TestLocation"},
			WindData: models.WindData{
				WindSpeed_10m:  func(v float64) *float64 { return &v }(3.0),
				WindSpeed_100m: func(v float64) *float64 { return &v }(4.0),
			},
		},
		{
			Time: time.Date(2022, 1, 1, 13, 0, 0, 0, time.UTC),
			Location: models.Location{Name: "TestLocation"},
			WindData: models.WindData{
				WindSpeed_10m:  func(v float64) *float64 { return &v }(2.5),
				WindSpeed_100m: func(v float64) *float64 { return &v }(3.5),
			},
		},
		{
			Time: time.Date(2022, 1, 1, 14, 0, 0, 0, time.UTC),
			Location: models.Location{Name: "TestLocation"},
			WindData: models.WindData{
				WindSpeed_10m:  func(v float64) *float64 { return &v }(0), // Invalid
				WindSpeed_100m: func(v float64) *float64 { return &v }(4.0),
			},
		},
	}

	results := CalculateHellmannExponentsForDataset(records)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that alpha values are reasonable
	for _, result := range results {
		if result.Alpha <= 0 || result.Alpha >= 1.0 {
			t.Errorf("Alpha should be between 0 and 1, got %v", result.Alpha)
		}
	}
}

func TestValidatePowerLawModel(t *testing.T) {
	// Create test data
	v10m := 3.0
	v100m := 4.0
	alpha := CalculateHellmannExponent(v10m, v100m, 10, 100)

	records := []models.WindRecord{
		{
			Time: time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
			Location: models.Location{Name: "TestLocation"},
			WindData: models.WindData{
				WindSpeed_10m:  func(v float64) *float64 { return &v }(v10m),
				WindSpeed_100m: func(v float64) *float64 { return &v }(v100m),
				WindSpeed_80m:  func(v float64) *float64 { return &v }(InterpolateWindSpeed(v10m, 10, 80, alpha)),
				WindSpeed_120m: func(v float64) *float64 { return &v }(InterpolateWindSpeed(v10m, 10, 120, alpha)),
			},
		},
	}

	exponents := []HellmannExponentResult{
		{
			Time:     time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
			Location: models.Location{Name: "TestLocation"},
			Alpha:    alpha,
		},
	}

	results := ValidatePowerLawModel(records, exponents)

	// Should have results for at least one height
	if len(results) == 0 {
		t.Error("Expected validation results, got none")
	}

	// Check that error metrics are reasonable
	for height, result := range results {
		if result.MAE < 0 {
			t.Errorf("MAE should be non-negative for height %v, got %v", height, result.MAE)
		}
		if result.RMSE < 0 {
			t.Errorf("RMSE should be non-negative for height %v, got %v", height, result.RMSE)
		}
		if result.SampleCount == 0 {
			t.Errorf("Sample count should be positive for height %v", height)
		}
	}
}

func TestCalculateMAE(t *testing.T) {
	predicted := []float64{1.0, 2.0, 3.0}
	actual := []float64{1.1, 2.1, 2.9}

	mae := calculateMAE(predicted, actual)
	expected := (0.1 + 0.1 + 0.1) / 3.0

	if math.Abs(mae-expected) > 1e-10 {
		t.Errorf("Expected %v, got %v", expected, mae)
	}
}

func TestCalculateRMSE(t *testing.T) {
	predicted := []float64{1.0, 2.0, 3.0}
	actual := []float64{1.0, 2.0, 3.0}

	rmse := calculateRMSE(predicted, actual)

	if rmse != 0 {
		t.Errorf("Expected 0 for perfect match, got %v", rmse)
	}
}

func TestCalculateCorrelation(t *testing.T) {
	// Perfect positive correlation
	predicted := []float64{1.0, 2.0, 3.0}
	actual := []float64{2.0, 4.0, 6.0}

	corr := calculateCorrelation(predicted, actual)

	if math.Abs(corr-1.0) > 1e-10 {
		t.Errorf("Expected correlation close to 1.0, got %v", corr)
	}
}

func TestGetWindSpeedAtHeight(t *testing.T) {
	windData := models.WindData{
		WindSpeed_10m:  func(v float64) *float64 { return &v }(3.0),
		WindSpeed_80m:  func(v float64) *float64 { return &v }(4.0),
		WindSpeed_100m: func(v float64) *float64 { return &v }(5.0),
	}

	// Test valid heights
	if *getWindSpeedAtHeight(windData, 10) != 3.0 {
		t.Error("Expected 3.0 for 10m height")
	}
	if *getWindSpeedAtHeight(windData, 80) != 4.0 {
		t.Error("Expected 4.0 for 80m height")
	}

	// Test invalid height
	if getWindSpeedAtHeight(windData, 50) != nil {
		t.Error("Expected nil for invalid height")
	}
}

func TestFilterRecordsByYear(t *testing.T) {
	records := []models.WindRecord{
		{Time: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Time: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Time: time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)},
		{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	filtered := FilterRecordsByYear(records, 2022)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 records for 2022, got %d", len(filtered))
	}

	for _, record := range filtered {
		if record.Time.Year() != 2022 {
			t.Errorf("Expected all records to be from 2022, got %d", record.Time.Year())
		}
	}
}

func TestFilterRecordsFromYear(t *testing.T) {
	records := []models.WindRecord{
		{Time: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Time: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	filtered := FilterRecordsFromYear(records, 2022)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 records from 2022 onwards, got %d", len(filtered))
	}

	for _, record := range filtered {
		if record.Time.Year() < 2022 {
			t.Errorf("Expected all records to be from 2022 or later, got %d", record.Time.Year())
		}
	}
}
