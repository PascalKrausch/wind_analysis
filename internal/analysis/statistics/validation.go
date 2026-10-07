package statistics

import (
	"math"

	"gonum.org/v1/gonum/stat"
)

func CalculateMAE(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}

	errs := make([]float64, len(predicted))
	for i := range predicted {
		errs[i] = math.Abs(predicted[i] - actual[i])
	}
	return stat.Mean(errs, nil)
}

func CalculateRMSE(predicted, actual []float64) float64 {
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

func CalculateCorrelation(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}
	return stat.Correlation(predicted, actual, nil)
}
