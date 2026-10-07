package server

import (
	"encoding/json"
	"testing"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/fitting"
	"wind_analysis/internal/analysis/statistics"
)

func TestDistributionHeightSpec(t *testing.T) {
	for _, height := range []int{10, 80, 100, 120, 180, 200} {
		spec, ok := distributionHeightSpec(height)
		if !ok || spec.HeightM != height || spec.Getter == nil {
			t.Errorf("distributionHeightSpec(%d) = (%+v, %t), want matching height spec", height, spec, ok)
		}
	}

	if _, ok := distributionHeightSpec(50); ok {
		t.Fatal("distributionHeightSpec(50) unexpectedly succeeded")
	}
}

func TestDistributionResponseFromFit(t *testing.T) {
	input := fitting.AnalysisPlotInput{
		LocationName: "Testort",
		HeightM:      100,
		FitterName:   "Weibull",
		Model:        distribution.Weibull{Shape: 2, Scale: 5, Location: 0},
		Metrics: distribution.FitMetrics{
			SampleSize:    12,
			LogLikelihood: -20,
			AIC:           46,
			BIC:           50,
			KSStatistic:   0.1,
			RMSE:          0.02,
		},
		Histogram:    []statistics.Bin{{Min: 0, Max: 1, Count: 3}},
		EmpiricalCDF: []statistics.CDFPoint{{X: 0.5, Y: 0.25}},
		FittedPDF:    []statistics.DensityPoint{{X: 0.5, Y: 0.1}},
		FittedCDF:    []statistics.CDFPoint{{X: 0.5, Y: 0.2}},
	}

	data, err := json.Marshal(distributionResponseFromFit(input))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var response map[string]json.RawMessage
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	for _, key := range []string{
		"location", "heightM", "modelName", "parameters", "sampleCount",
		"logLikelihood", "aic", "bic", "ksStatistic", "rmse",
		"histogram", "empiricalCDF", "fittedPDF", "fittedCDF",
	} {
		if _, ok := response[key]; !ok {
			t.Errorf("distribution response is missing JSON field %q", key)
		}
	}
}
