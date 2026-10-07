package statistics

import (
	"math"
	"testing"
)

func TestCalculateMAE(t *testing.T) {
	predicted := []float64{1.0, 2.0, 3.0}
	actual := []float64{1.1, 2.1, 2.9}

	mae := CalculateMAE(predicted, actual)
	expected := (0.1 + 0.1 + 0.1) / 3.0

	if math.Abs(mae-expected) > 1e-10 {
		t.Errorf("Expected %v, got %v", expected, mae)
	}
}

func TestCalculateRMSE(t *testing.T) {
	predicted := []float64{1.0, 2.0, 3.0}
	actual := []float64{1.0, 2.0, 3.0}

	rmse := CalculateRMSE(predicted, actual)

	if rmse != 0 {
		t.Errorf("Expected 0 for perfect match, got %v", rmse)
	}
}

func TestCalculateCorrelation(t *testing.T) {
	// Perfect positive correlation
	predicted := []float64{1.0, 2.0, 3.0}
	actual := []float64{2.0, 4.0, 6.0}

	corr := CalculateCorrelation(predicted, actual)

	if math.Abs(corr-1.0) > 1e-10 {
		t.Errorf("Expected correlation close to 1.0, got %v", corr)
	}
}
