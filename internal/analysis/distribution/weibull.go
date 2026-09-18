package distribution

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/stat"
)

// Weibull repräsentiert eine Weibull-Verteilung mit Shape-, Scale- und Location-Parametern.
type Weibull struct {
	Shape    float64 // Formparameter (k)
	Scale    float64 // Skalenparameter (λ)
	Location float64 // Lageparameter (μ)
}

func (w Weibull) Params() []float64 {
	return []float64{w.Shape, w.Scale, w.Location}
}

func (w Weibull) Valid() bool {
	return IsFinite(w.Shape) && IsFinite(w.Scale) && IsFinite(w.Location) && w.Shape > 0 && w.Scale > 0
}

func (w Weibull) PDF(x float64) float64 {
	if !w.Valid() {
		return 0
	}
	z := (x - w.Location) / w.Scale
	if z < 0 {
		return 0
	}
	if z == 0 {
		switch {
		case w.Shape < 1:
			return math.Inf(1)
		case w.Shape == 1:
			return w.Shape / w.Scale
		default:
			return 0
		}
	}

	return (w.Shape / w.Scale) *
		math.Pow(z, w.Shape-1) *
		math.Exp(-math.Pow(z, w.Shape))
}

func (w Weibull) CDF(x float64) float64 {
	if !w.Valid() {
		return 0
	}
	z := (x - w.Location) / w.Scale
	if z < 0 {
		return 0
	}
	return 1 - math.Exp(-math.Pow(z, w.Shape))
}

func (w Weibull) LogPDF(x float64) float64 {
	if !w.Valid() {
		return math.Inf(-1)
	}
	z := (x - w.Location) / w.Scale
	if z < 0 {
		return math.Inf(-1)
	}
	if z == 0 {
		switch {
		case w.Shape < 1:
			return math.Inf(1)
		case w.Shape == 1:
			return math.Log(w.Shape / w.Scale)
		default:
			return math.Inf(-1)
		}
	}

	return math.Log(w.Shape/w.Scale) +
		(w.Shape-1)*math.Log(z) -
		math.Pow(z, w.Shape)
}

// WeibullParams repräsentiert die Parameter der Weibull-Verteilung (legacy, für Kompatibilität).
type WeibullParams struct {
	Shape    float64 // Formparameter (k)
	Scale    float64 // Skalenparameter (λ)
	Location float64 // Lageparameter (μ)
}

func (p WeibullParams) Valid() bool {
	return IsFinite(p.Shape) && IsFinite(p.Scale) && IsFinite(p.Location) && p.Shape > 0 && p.Scale > 0
}

// Legacy Typen für Kompatibilität mit existierendem Code
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

// WeibullPDF berechnet die Wahrscheinlichkeitsdichtefunktion (PDF) - legacy Funktion.
func WeibullPDF(x float64, params WeibullParams) float64 {
	w := Weibull{Shape: params.Shape, Scale: params.Scale, Location: params.Location}
	return w.PDF(x)
}

// WeibullCDF berechnet die kumulative Verteilungsfunktion (CDF) - legacy Funktion.
func WeibullCDF(x float64, params WeibullParams) float64 {
	w := Weibull{Shape: params.Shape, Scale: params.Scale, Location: params.Location}
	return w.CDF(x)
}

func WeibullLogPDF(x float64, params WeibullParams) float64 {
	w := Weibull{Shape: params.Shape, Scale: params.Scale, Location: params.Location}
	return w.LogPDF(x)
}

// WeibullFitter schätzt Shape, Scale und Location per
// lokationsgestützter Maximum-Likelihood-Näherung.
type WeibullFitter struct{}

func (f WeibullFitter) Name() string {
	return "Weibull"
}

func (f WeibullFitter) Fit(data []float64) (ContinuousDistribution, error) {
	params, err := estimateWeibullParameters(data)
	if err != nil {
		return nil, err
	}
	return Weibull{
		Shape:    params.Shape,
		Scale:    params.Scale,
		Location: params.Location,
	}, nil
}

// EstimateWeibullParameters schätzt Shape, Scale und Location per
// lokationsgestützter Maximum-Likelihood-Näherung (legacy).
func EstimateWeibullParameters(data []float64) (WeibullParams, error) {
	return estimateWeibullParameters(data)
}

func estimateWeibullParameters(data []float64) (WeibullParams, error) {
	cleaned := CleanData(data)
	if len(cleaned) < 3 {
		return WeibullParams{}, errors.New("zu wenige gültige Datenpunkte")
	}

	summary := Summarize(cleaned)
	if summary.Min == summary.Max {
		loc := summary.Min - 1e-6
		return WeibullParams{
			Shape:    1,
			Scale:    math.Max(summary.Min-loc, 1e-6),
			Location: loc,
		}, nil
	}

	searchWidth := math.Max(3*summary.StdDev, math.Max(0.1*(summary.Max-summary.Min), 1.0))
	searchLower := math.Min(0, summary.Min-searchWidth)
	searchUpper := summary.Min - 1e-9
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
	cleaned := CleanData(data)

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
	w := Weibull{Shape: params.Shape, Scale: params.Scale, Location: params.Location}
	cleaned := CleanData(data)
	n := len(cleaned)
	if n == 0 || !w.Valid() {
		return WeibullFitMetrics{}
	}

	sorted := SortedCopy(cleaned)
	ll := WeibullLogLikelihood(cleaned, params)

	aic := 2*3.0 - 2*ll
	bic := math.Log(float64(n))*3.0 - 2*ll

	ks := 0.0
	mse := 0.0
	for i, x := range sorted {
		model := w.CDF(x)
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
	if !IsFinite(minX) || !IsFinite(maxX) || maxX <= minX {
		maxX = minX + 1
	}

	maxDiff := 0.0
	meanAbsDiff := 0.0

	leftDist := Weibull{Shape: left.Shape, Scale: left.Scale, Location: left.Location}
	rightDist := Weibull{Shape: right.Shape, Scale: right.Scale, Location: right.Location}

	for i := 0; i < grid; i++ {
		x := minX + (float64(i)/float64(grid-1))*(maxX-minX)
		diff := math.Abs(leftDist.CDF(x) - rightDist.CDF(x))
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
	w := Weibull{Shape: params.Shape, Scale: params.Scale, Location: params.Location}
	if !w.Valid() {
		return math.Inf(-1)
	}
	ll := 0.0
	for _, x := range data {
		v := w.LogPDF(x)
		if math.IsInf(v, -1) {
			return math.Inf(-1)
		}
		ll += v
	}
	return ll
}

func fitWeibullAtLocation(data []float64, location float64) (WeibullParams, float64, bool) {
	shifted := make([]float64, len(data))
	logs := make([]float64, len(data))

	for i, x := range data {
		y := x - location
		if y <= 0 || !IsFinite(y) {
			return WeibullParams{}, math.Inf(-1), false
		}
		shifted[i] = y
		logs[i] = math.Log(y)
	}

	summary := Summarize(shifted)
	if summary.Mean <= 0 || !IsFinite(summary.Mean) {
		return WeibullParams{}, math.Inf(-1), false
	}

	shape := math.Pow(summary.StdDev/summary.Mean, -1.086)
	if !IsFinite(shape) || shape <= 0 {
		shape = 1.5
	}
	shape = refineWeibullShape(logs, shape)
	if !IsFinite(shape) || shape <= 0 {
		return WeibullParams{}, math.Inf(-1), false
	}

	scale := weibullScaleFromLogs(logs, shape)
	if !IsFinite(scale) || scale <= 0 {
		return WeibullParams{}, math.Inf(-1), false
	}

	params := WeibullParams{Shape: shape, Scale: scale, Location: location}
	w := Weibull{Shape: shape, Scale: scale, Location: location}
	ll := 0.0
	for _, x := range data {
		v := w.LogPDF(x)
		if math.IsInf(v, -1) {
			return WeibullParams{}, math.Inf(-1), false
		}
		ll += v
	}
	if !IsFinite(ll) {
		return WeibullParams{}, math.Inf(-1), false
	}

	return params, ll, true
}

func refineWeibullShape(logs []float64, initial float64) float64 {
	k := initial
	if !IsFinite(k) || k <= 0 {
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

		if sumPow <= 0 || !IsFinite(sumPow) {
			break
		}

		denom := sumPowLog/sumPow - meanLog
		if denom <= 0 || !IsFinite(denom) {
			break
		}

		next := 1.0 / denom
		if !IsFinite(next) || next <= 0 {
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
