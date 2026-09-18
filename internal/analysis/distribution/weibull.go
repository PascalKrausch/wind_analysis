package distribution

import (
	"errors"
	"math"
	"sort"

	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/stat"
)

// WeibullParams repräsentiert die Parameter der Weibull-Verteilung.
type WeibullParams struct {
	Shape    float64 // Formparameter (k)
	Scale    float64 // Skalenparameter (λ)
	Location float64 // Lageparameter (μ)
}

type WeibullFitMetrics struct {
	LogLikelihood float64
	AIC           float64
	BIC           float64
	KSStatistic   float64
	RMSE          float64
}

type WeibullAnalysisResult struct {
	SeriesName    string
	HeightM       int
	SampleSize    int
	Shape         float64
	Scale         float64
	Location      float64
	LogLikelihood float64
	AIC           float64
	BIC           float64
	KSStatistic   float64
	RMSE          float64
}

type WeibullComparisonResult struct {
	LeftSeriesName  string
	RightSeriesName string
	LeftHeightM     int
	RightHeightM    int
	ShapeDelta      float64
	ScaleDelta      float64
	LocationDelta   float64
	KSStatistic     float64
	MeanAbsCDFDiff  float64
}

func (p WeibullParams) Valid() bool {
	return isFinite(p.Shape) && isFinite(p.Scale) && isFinite(p.Location) && p.Shape > 0 && p.Scale > 0
}

// WeibullPDF berechnet die Wahrscheinlichkeitsdichtefunktion (PDF).
func WeibullPDF(x float64, params WeibullParams) float64 {
	if !params.Valid() {
		return 0
	}
	z := (x - params.Location) / params.Scale
	if z < 0 {
		return 0
	}
	if z == 0 {
		switch {
		case params.Shape < 1:
			return math.Inf(1)
		case params.Shape == 1:
			return params.Shape / params.Scale
		default:
			return 0
		}
	}

	return (params.Shape / params.Scale) *
		math.Pow(z, params.Shape-1) *
		math.Exp(-math.Pow(z, params.Shape))
}

// WeibullCDF berechnet die kumulative Verteilungsfunktion (CDF).
func WeibullCDF(x float64, params WeibullParams) float64 {
	if !params.Valid() {
		return 0
	}
	z := (x - params.Location) / params.Scale
	if z < 0 {
		return 0
	}
	return 1 - math.Exp(-math.Pow(z, params.Shape))
}

func WeibullLogPDF(x float64, params WeibullParams) float64 {
	if !params.Valid() {
		return math.Inf(-1)
	}
	z := (x - params.Location) / params.Scale
	if z < 0 {
		return math.Inf(-1)
	}
	if z == 0 {
		switch {
		case params.Shape < 1:
			return math.Inf(1)
		case params.Shape == 1:
			return math.Log(params.Shape / params.Scale)
		default:
			return math.Inf(-1)
		}
	}

	return math.Log(params.Shape/params.Scale) +
		(params.Shape-1)*math.Log(z) -
		math.Pow(z, params.Shape)
}

// EstimateWeibullParameters schätzt Shape, Scale und Location per
// lokationsgestützter Maximum-Likelihood-Näherung.
func EstimateWeibullParameters(data []float64) (WeibullParams, error) {
	cleaned := cleanWindSpeedData(data)
	if len(cleaned) < 3 {
		return WeibullParams{}, errors.New("zu wenige gültige Datenpunkte")
	}

	summary := summarize(cleaned)
	if summary.min == summary.max {
		loc := summary.min - 1e-6
		return WeibullParams{
			Shape:    1,
			Scale:    math.Max(summary.min-loc, 1e-6),
			Location: loc,
		}, nil
	}

	searchWidth := math.Max(3*summary.stdDev, math.Max(0.1*(summary.max-summary.min), 1.0))
	searchLower := math.Min(0, summary.min-searchWidth)
	searchUpper := summary.min - 1e-9
	if searchLower >= searchUpper {
		searchLower = searchUpper - math.Max(searchWidth, 1.0)
	}

	bestLL := math.Inf(-1)
	best := WeibullParams{}
	found := false

	const candidateCount = 32
	for i := 0; i < candidateCount; i++ {
		loc := searchLower + float64(i)*(searchUpper-searchLower)/float64(candidateCount-1)
		params, ll, ok := fitWeibullAtLocation(cleaned, loc)
		if ok && ll > bestLL {
			bestLL = ll
			best = params
			found = true
		}
	}

	if !found {
		return WeibullParams{}, errors.New("weibull fit fehlgeschlagen")
	}

	return best, nil
}

func AnalyzeWeibullSeries(seriesName string, heightM int, data []float64) (WeibullAnalysisResult, error) {
	params, err := EstimateWeibullParameters(data)
	if err != nil {
		return WeibullAnalysisResult{}, err
	}

	metrics := ValidateWeibullFit(data, params)
	cleaned := cleanWindSpeedData(data)

	return WeibullAnalysisResult{
		SeriesName:    seriesName,
		HeightM:       heightM,
		SampleSize:    len(cleaned),
		Shape:         params.Shape,
		Scale:         params.Scale,
		Location:      params.Location,
		LogLikelihood: metrics.LogLikelihood,
		AIC:           metrics.AIC,
		BIC:           metrics.BIC,
		KSStatistic:   metrics.KSStatistic,
		RMSE:          metrics.RMSE,
	}, nil
}

func ValidateWeibullFit(data []float64, params WeibullParams) WeibullFitMetrics {
	cleaned := cleanWindSpeedData(data)
	n := len(cleaned)
	if n == 0 || !params.Valid() {
		return WeibullFitMetrics{}
	}

	sorted := sortedCopy(cleaned)
	ll := WeibullLogLikelihood(cleaned, params)

	aic := 2*3.0 - 2*ll
	bic := math.Log(float64(n))*3.0 - 2*ll

	ks := 0.0
	mse := 0.0
	for i, x := range sorted {
		model := WeibullCDF(x, params)
		upper := float64(i+1) / float64(n)
		lower := float64(i) / float64(n)

		d := math.Max(math.Abs(upper-model), math.Abs(model-lower))
		if d > ks {
			ks = d
		}

		err := model - upper
		mse += err * err
	}

	return WeibullFitMetrics{
		LogLikelihood: ll,
		AIC:           aic,
		BIC:           bic,
		KSStatistic:   ks,
		RMSE:          math.Sqrt(mse / float64(n)),
	}
}

func CompareWeibullSeries(left, right WeibullAnalysisResult) WeibullComparisonResult {
	grid := 128
	minX := math.Min(left.Location, right.Location)
	maxX := math.Max(left.Location+8*left.Scale, right.Location+8*right.Scale)
	if !isFinite(minX) || !isFinite(maxX) || maxX <= minX {
		maxX = minX + 1
	}

	maxDiff := 0.0
	meanAbsDiff := 0.0

	leftParams := WeibullParams{Shape: left.Shape, Scale: left.Scale, Location: left.Location}
	rightParams := WeibullParams{Shape: right.Shape, Scale: right.Scale, Location: right.Location}

	for i := 0; i < grid; i++ {
		x := minX + (float64(i)/float64(grid-1))*(maxX-minX)
		diff := math.Abs(WeibullCDF(x, leftParams) - WeibullCDF(x, rightParams))
		if diff > maxDiff {
			maxDiff = diff
		}
		meanAbsDiff += diff
	}

	return WeibullComparisonResult{
		LeftSeriesName:  left.SeriesName,
		RightSeriesName: right.SeriesName,
		LeftHeightM:     left.HeightM,
		RightHeightM:    right.HeightM,
		ShapeDelta:      left.Shape - right.Shape,
		ScaleDelta:      left.Scale - right.Scale,
		LocationDelta:   left.Location - right.Location,
		KSStatistic:     maxDiff,
		MeanAbsCDFDiff:  meanAbsDiff / float64(grid),
	}
}

func WeibullLogLikelihood(data []float64, params WeibullParams) float64 {
	if !params.Valid() {
		return math.Inf(-1)
	}
	ll := 0.0
	for _, x := range data {
		v := WeibullLogPDF(x, params)
		if math.IsInf(v, -1) {
			return math.Inf(-1)
		}
		ll += v
	}
	return ll
}

type sampleSummary struct {
	mean   float64
	stdDev float64
	min    float64
	max    float64
}

func cleanWindSpeedData(data []float64) []float64 {
	out := make([]float64, 0, len(data))
	for _, v := range data {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			continue
		}
		out = append(out, v)
	}
	return out
}

func summarize(data []float64) sampleSummary {
	n := len(data)
	if n == 0 {
		return sampleSummary{}
	}

	summary := sampleSummary{
		mean: stat.Mean(data, nil),
		min:  floats.Min(data),
		max:  floats.Max(data),
	}
	if n > 1 {
		variance := stat.Variance(data, nil)
		if variance > 0 {
			summary.stdDev = math.Sqrt(variance)
		}
	}
	return summary
}

func sortedCopy(data []float64) []float64 {
	out := make([]float64, len(data))
	copy(out, data)
	sort.Float64s(out)
	return out
}

func fitWeibullAtLocation(data []float64, location float64) (WeibullParams, float64, bool) {
	shifted := make([]float64, len(data))
	logs := make([]float64, len(data))

	for i, x := range data {
		y := x - location
		if y <= 0 || !isFinite(y) {
			return WeibullParams{}, math.Inf(-1), false
		}
		shifted[i] = y
		logs[i] = math.Log(y)
	}

	summary := summarize(shifted)
	if summary.mean <= 0 || !isFinite(summary.mean) {
		return WeibullParams{}, math.Inf(-1), false
	}

	shape := math.Pow(summary.stdDev/summary.mean, -1.086)
	if !isFinite(shape) || shape <= 0 {
		shape = 1.5
	}
	shape = refineWeibullShape(logs, shape)
	if !isFinite(shape) || shape <= 0 {
		return WeibullParams{}, math.Inf(-1), false
	}

	scale := weibullScaleFromLogs(logs, shape)
	if !isFinite(scale) || scale <= 0 {
		return WeibullParams{}, math.Inf(-1), false
	}

	params := WeibullParams{Shape: shape, Scale: scale, Location: location}
	ll := WeibullLogLikelihood(data, params)
	if !isFinite(ll) {
		return WeibullParams{}, math.Inf(-1), false
	}

	return params, ll, true
}

func refineWeibullShape(logs []float64, initial float64) float64 {
	k := initial
	if !isFinite(k) || k <= 0 {
		k = 1.5
	}

	meanLog := stat.Mean(logs, nil)
	for iter := 0; iter < 64; iter++ {
		sumPow := 0.0
		sumPowLog := 0.0
		for _, logY := range logs {
			pow := math.Exp(k * logY)
			sumPow += pow
			sumPowLog += pow * logY
		}

		if sumPow <= 0 || !isFinite(sumPow) {
			break
		}

		denom := sumPowLog/sumPow - meanLog
		if denom <= 0 || !isFinite(denom) {
			break
		}

		next := 1.0 / denom
		if !isFinite(next) || next <= 0 {
			break
		}

		if math.Abs(next-k) <= 1e-7*math.Max(1, k) {
			k = next
			break
		}
		k = next
	}

	return math.Max(k, 1e-6)
}

func weibullScaleFromLogs(logs []float64, shape float64) float64 {
	sumPow := 0.0
	for _, logY := range logs {
		sumPow += math.Exp(shape * logY)
	}
	return math.Pow(sumPow/float64(len(logs)), 1.0/shape)
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
