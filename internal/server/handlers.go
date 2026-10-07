package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"wind_analysis/internal/analysis/distribution"
	"wind_analysis/internal/analysis/fitting"
	"wind_analysis/internal/analysis/statistics"
	"wind_analysis/models"
)

type LocationResponse struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

func (s *Server) handleLocations(w http.ResponseWriter, r *http.Request) {
	config, err := loadConfig("config.yaml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	locations := make([]LocationResponse, len(config.LocationList))
	for i, loc := range config.LocationList {
		locations[i] = LocationResponse{
			Name: loc.Name,
			Lat:  loc.Latitude,
			Lon:  loc.Longitude,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locations)
}

type TimeSeriesRequest struct {
	Locations []string `json:"locations"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	Metric    string   `json:"metric"` // "wind" oder "hellmann"
}

type TimeSeriesPoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

func (s *Server) handleTimeSeries(w http.ResponseWriter, r *http.Request) {
	var req TimeSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := make(map[string][]TimeSeriesPoint)

	for _, locName := range req.Locations {
		records, err := s.db.LoadWindData(r.Context(), locName, req.StartDate, req.EndDate)
		if err != nil {
			continue
		}

		points := downsampleRecords(records, req.Metric)
		result[locName] = points
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type DistributionRequest struct {
	Locations []string `json:"locations"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	HeightM   int      `json:"heightM"`
}

type DistributionPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type DistributionBin struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
	Count int     `json:"count"`
}

type DistributionResponse struct {
	Location      string              `json:"location"`
	HeightM       int                 `json:"heightM"`
	ModelName     string              `json:"modelName"`
	Parameters    []float64           `json:"parameters"`
	SampleCount   int                 `json:"sampleCount"`
	LogLikelihood float64             `json:"logLikelihood"`
	AIC           float64             `json:"aic"`
	BIC           float64             `json:"bic"`
	KSStatistic   float64             `json:"ksStatistic"`
	RMSE          float64             `json:"rmse"`
	Histogram     []DistributionBin   `json:"histogram"`
	EmpiricalCDF  []DistributionPoint `json:"empiricalCDF"`
	FittedPDF     []DistributionPoint `json:"fittedPDF"`
	FittedCDF     []DistributionPoint `json:"fittedCDF"`
}

func (s *Server) handleDistributions(w http.ResponseWriter, r *http.Request) {
	var req DistributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		http.Error(w, "Ungültiges Startdatum; erwartet wird YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil || endDate.Before(startDate) {
		http.Error(w, "Ungültiges Enddatum; es muss im Format YYYY-MM-DD und nicht vor dem Startdatum liegen", http.StatusBadRequest)
		return
	}
	spec, ok := distributionHeightSpec(req.HeightM)
	if !ok {
		http.Error(w, "Ungültige Höhe; erlaubt sind 10, 80, 100, 120, 180 oder 200 Meter", http.StatusBadRequest)
		return
	}
	if len(req.Locations) == 0 {
		http.Error(w, "Mindestens ein Standort muss ausgewählt werden", http.StatusBadRequest)
		return
	}

	fitters := []distribution.Fitter{
		distribution.WeibullFitter{},
		distribution.LogNormalFitter{},
		distribution.GammaFitter{},
	}
	results := make([]DistributionResponse, 0, len(req.Locations))
	for _, location := range req.Locations {
		records, err := s.db.LoadWindData(r.Context(), location, req.StartDate, req.EndDate)
		if err != nil {
			http.Error(w, fmt.Sprintf("Winddaten für %s konnten nicht geladen werden: %v", location, err), http.StatusInternalServerError)
			return
		}

		inputs, err := fitting.FindBestFitsForLocation(location, records, fitters, []fitting.HeightSpec{spec})
		if err != nil {
			http.Error(w, fmt.Sprintf("Verteilungsanalyse für %s fehlgeschlagen: %v", location, err), http.StatusUnprocessableEntity)
			return
		}
		for _, input := range inputs {
			results = append(results, distributionResponseFromFit(input))
		}
	}

	data, err := json.Marshal(results)
	if err != nil {
		http.Error(w, "Verteilungsanalyse konnte nicht serialisiert werden", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func distributionHeightSpec(height int) (fitting.HeightSpec, bool) {
	for _, spec := range fitting.DefaultHeightSpecs {
		if spec.HeightM == height {
			return spec, true
		}
	}
	return fitting.HeightSpec{}, false
}

func distributionResponseFromFit(input fitting.AnalysisPlotInput) DistributionResponse {
	response := DistributionResponse{
		Location:      input.LocationName,
		HeightM:       input.HeightM,
		ModelName:     input.FitterName,
		Parameters:    input.Model.Params(),
		SampleCount:   input.Metrics.SampleSize,
		LogLikelihood: input.Metrics.LogLikelihood,
		AIC:           input.Metrics.AIC,
		BIC:           input.Metrics.BIC,
		KSStatistic:   input.Metrics.KSStatistic,
		RMSE:          input.Metrics.RMSE,
		Histogram:     make([]DistributionBin, 0, len(input.Histogram)),
		EmpiricalCDF:  mapCDFPoints(input.EmpiricalCDF),
		FittedPDF:     mapDensityPoints(input.FittedPDF),
		FittedCDF:     mapCDFPoints(input.FittedCDF),
	}
	for _, bin := range input.Histogram {
		response.Histogram = append(response.Histogram, DistributionBin{
			Lower: bin.Min,
			Upper: bin.Max,
			Count: bin.Count,
		})
	}
	return response
}

func mapDensityPoints(points []statistics.DensityPoint) []DistributionPoint {
	result := make([]DistributionPoint, len(points))
	for i, point := range points {
		result[i] = DistributionPoint{X: point.X, Y: point.Y}
	}
	return result
}

func mapCDFPoints(points []statistics.CDFPoint) []DistributionPoint {
	result := make([]DistributionPoint, len(points))
	for i, point := range points {
		result[i] = DistributionPoint{X: point.X, Y: point.Y}
	}
	return result
}

type ValidationRequest struct {
	Locations []string `json:"locations"`
}

type ValidationResponse struct {
	Location    string  `json:"location"`
	HeightM     float64 `json:"heightM"`
	MAE         float64 `json:"mae"`
	RMSE        float64 `json:"rmse"`
	Correlation float64 `json:"correlation"`
	SampleCount int     `json:"sampleCount"`
}

func (s *Server) handleValidation(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := make([]ValidationResponse, 0)

	for _, locName := range req.Locations {
		// Lade Daten ab 2022 für Validierung
		records, err := s.db.LoadWindData(r.Context(), locName, "2022-01-01", time.Now().Format("2006-01-02"))
		if err != nil {
			continue
		}

		if len(records) == 0 {
			continue
		}

		// Berechne Hellmann-Exponenten
		exponents := calculateHellmannExponentsForRecords(records)
		if len(exponents) == 0 {
			continue
		}

		// Validierung für verschiedene Höhen
		validationResults := validatePowerLawForLocation(records, exponents)

		for height, result := range validationResults {
			results = append(results, ValidationResponse{
				Location:    locName,
				HeightM:     height,
				MAE:         result.MAE,
				RMSE:        result.RMSE,
				Correlation: result.Correlation,
				SampleCount: result.SampleCount,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

type HellmannDistributionRequest struct {
	Locations []string `json:"locations"`
	StartDate string   `json:"startDate"` // YYYY-MM-DD
	EndDate   string   `json:"endDate"`   // YYYY-MM-DD
	GroupBy   string   `json:"groupBy"`   // optional: "location" oder "all" (default: "all")
}

type HellmannDistributionResponse struct {
	Location      string              `json:"location"` // leer bei GroupBy="all"
	ModelName     string              `json:"modelName"`
	Parameters    []float64           `json:"parameters"`
	SampleCount   int                 `json:"sampleCount"`
	LogLikelihood float64             `json:"logLikelihood"`
	AIC           float64             `json:"aic"`
	BIC           float64             `json:"bic"`
	KSStatistic   float64             `json:"ksStatistic"`
	RMSE          float64             `json:"rmse"`
	Histogram     []DistributionBin   `json:"histogram"`
	EmpiricalCDF  []DistributionPoint `json:"empiricalCDF"`
	FittedPDF     []DistributionPoint `json:"fittedPDF"`
	FittedCDF     []DistributionPoint `json:"fittedCDF"`
}

func (s *Server) handleHellmannDistributions(w http.ResponseWriter, r *http.Request) {
	var req HellmannDistributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validierung
	if err := validateHellmannRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Berechnung (delegiert an analysis layer)
	results := make([]HellmannDistributionResponse, 0)
	for _, location := range req.Locations {
		records, err := s.db.LoadWindData(r.Context(), location, req.StartDate, req.EndDate)
		if err != nil {
			continue // oder error je nach Anforderung
		}

		// Aufruf der Business-Logik
		fitters := []distribution.Fitter{
			distribution.WeibullFitter{},
			distribution.LogNormalFitter{},
			distribution.GammaFitter{},
		}
		fitResult, err := fitting.FitHellmannDistribution(location, records, fitters)
		if err != nil {
			continue
		}

		// Response formatieren
		results = append(results, hellmannResponseFromFit(fitResult))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func validateHellmannRequest(req HellmannDistributionRequest) error {
	if len(req.Locations) == 0 {
		return fmt.Errorf("mindestens ein Standort muss ausgewählt werden")
	}

	if req.StartDate == "" {
		return fmt.Errorf("Startdatum ist erforderlich")
	}

	if req.EndDate == "" {
		return fmt.Errorf("Enddatum ist erforderlich")
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return fmt.Errorf("ungültiges Startdatum; erwartet wird YYYY-MM-DD")
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return fmt.Errorf("ungültiges Enddatum; erwartet wird YYYY-MM-DD")
	}

	if endDate.Before(startDate) {
		return fmt.Errorf("Enddatum muss nach dem Startdatum liegen")
	}

	if req.GroupBy != "" && req.GroupBy != "location" && req.GroupBy != "all" {
		return fmt.Errorf("groupBy muss entweder 'location' oder 'all' sein")
	}

	return nil
}

func hellmannResponseFromFit(fitResult fitting.HellmannFitResult) HellmannDistributionResponse {
	// Extrahiere Modellnamen aus dem Verteilungstyp durch Type Assertion
	modelName := getDistributionName(fitResult.Model)

	response := HellmannDistributionResponse{
		Location:      fitResult.LocationName,
		ModelName:     modelName,
		Parameters:    fitResult.Model.Params(),
		SampleCount:   fitResult.Metrics.SampleSize,
		LogLikelihood: fitResult.Metrics.LogLikelihood,
		AIC:           fitResult.Metrics.AIC,
		BIC:           fitResult.Metrics.BIC,
		KSStatistic:   fitResult.Metrics.KSStatistic,
		RMSE:          fitResult.Metrics.RMSE,
		Histogram:     make([]DistributionBin, 0, len(fitResult.Histogram)),
		EmpiricalCDF:  mapCDFPoints(fitResult.EmpiricalCDF),
		FittedPDF:     mapDensityPoints(fitResult.FittedPDF),
		FittedCDF:     mapCDFPoints(fitResult.FittedCDF),
	}
	for _, bin := range fitResult.Histogram {
		response.Histogram = append(response.Histogram, DistributionBin{
			Lower: bin.Min,
			Upper: bin.Max,
			Count: bin.Count,
		})
	}
	return response
}

func getDistributionName(dist distribution.ContinuousDistribution) string {
	switch dist.(type) {
	case distribution.Weibull:
		return "Weibull"
	case distribution.LogNormal:
		return "LogNormal"
	case distribution.Gamma:
		return "Gamma"
	case distribution.Normal:
		return "Normal"
	case distribution.Beta:
		return "Beta"
	default:
		return "Unknown"
	}
}

type HellmannExponentResult struct {
	Time  time.Time
	Alpha float64
}

func calculateHellmannExponentsForRecords(records []models.WindRecord) []HellmannExponentResult {
	var results []HellmannExponentResult

	for _, record := range records {
		alpha, ok := calculateHellmannExponent(record)
		if ok {
			results = append(results, HellmannExponentResult{
				Time:  record.Time,
				Alpha: alpha,
			})
		}
	}

	return results
}

type ValidationResult struct {
	Height      float64
	MAE         float64
	RMSE        float64
	Correlation float64
	SampleCount int
}

func validatePowerLawForLocation(records []models.WindRecord, exponents []HellmannExponentResult) map[float64]ValidationResult {
	// Exponent Map für schnellen Lookup
	exponentMap := make(map[string]float64)
	for _, exp := range exponents {
		key := exp.Time.Format(time.RFC3339Nano)
		exponentMap[key] = exp.Alpha
	}

	// Validierungshöhen
	heights := []float64{80, 120, 180, 200}

	// Stats pro Höhe
	statsByHeight := make(map[float64]struct {
		predicted []float64
		actual    []float64
	})

	for _, h := range heights {
		statsByHeight[h] = struct {
			predicted []float64
			actual    []float64
		}{predicted: []float64{}, actual: []float64{}}
	}

	for _, record := range records {
		key := record.Time.Format(time.RFC3339Nano)
		alpha, exists := exponentMap[key]
		if !exists || math.IsNaN(alpha) {
			continue
		}

		v10m, ok := extractWindSpeed(record)
		if !ok || v10m <= 0 {
			continue
		}

		for _, height := range heights {
			vActual := extractWindSpeedAtHeight(record, int(height))
			if vActual <= 0 {
				continue
			}

			// Interpoliere mittels Hellmann-Exponent
			vPredicted := v10m * math.Pow(height/10.0, alpha)
			if math.IsNaN(vPredicted) {
				continue
			}

			stats := statsByHeight[height]
			stats.predicted = append(stats.predicted, vPredicted)
			stats.actual = append(stats.actual, vActual)
			statsByHeight[height] = stats
		}
	}

	// Berechne Metriken
	results := make(map[float64]ValidationResult)
	for height, stats := range statsByHeight {
		if len(stats.predicted) == 0 {
			continue
		}

		mae := calculateMAE(stats.predicted, stats.actual)
		rmse := calculateRMSE(stats.predicted, stats.actual)
		corr := calculateCorrelation(stats.predicted, stats.actual)

		results[height] = ValidationResult{
			Height:      height,
			MAE:         mae,
			RMSE:        rmse,
			Correlation: corr,
			SampleCount: len(stats.predicted),
		}
	}

	return results
}

func calculateMAE(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}

	sum := 0.0
	for i := range predicted {
		sum += math.Abs(predicted[i] - actual[i])
	}
	return sum / float64(len(predicted))
}

func calculateRMSE(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}

	sum := 0.0
	for i := range predicted {
		diff := predicted[i] - actual[i]
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(predicted)))
}

func calculateCorrelation(predicted, actual []float64) float64 {
	if len(predicted) != len(actual) || len(predicted) == 0 {
		return math.NaN()
	}

	n := float64(len(predicted))

	// Mittelwerte
	meanPred := 0.0
	meanAct := 0.0
	for i := range predicted {
		meanPred += predicted[i]
		meanAct += actual[i]
	}
	meanPred /= n
	meanAct /= n

	// Kovarianz und Varianzen
	cov := 0.0
	varPred := 0.0
	varAct := 0.0

	for i := range predicted {
		dp := predicted[i] - meanPred
		da := actual[i] - meanAct
		cov += dp * da
		varPred += dp * dp
		varAct += da * da
	}

	if varPred == 0 || varAct == 0 {
		return math.NaN()
	}

	return cov / (math.Sqrt(varPred) * math.Sqrt(varAct))
}

func loadConfig(configPath string) (*models.Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func downsampleRecords(records []models.WindRecord, metric string) []TimeSeriesPoint {
	if len(records) == 0 {
		return []TimeSeriesPoint{}
	}

	// Zeitbereich berechnen
	var start, end time.Time
	for _, record := range records {
		if !record.Time.IsZero() {
			if start.IsZero() || record.Time.Before(start) {
				start = record.Time
			}
			if end.IsZero() || record.Time.After(end) {
				end = record.Time
			}
		}
	}

	if start.IsZero() || end.IsZero() {
		return []TimeSeriesPoint{}
	}

	totalDays := end.Sub(start).Hours() / 24
	locationCount := 1

	// Downsampling basierend auf Zeitraum
	sampleInterval := 1
	if totalDays > 365 {
		sampleInterval = 48
	} else if totalDays > 90 {
		sampleInterval = 24
	} else if totalDays > 30 {
		sampleInterval = 12
	}

	// Zusätzliches Scaling basierend auf Standort-Anzahl
	if locationCount > 10 {
		sampleInterval = sampleInterval * 2
	} else if locationCount > 5 {
		sampleInterval = sampleInterval * 3 / 2
	}

	if sampleInterval < 1 {
		sampleInterval = 1
	}

	points := make([]TimeSeriesPoint, 0)
	for i, record := range records {
		if i%sampleInterval == 0 {
			var value float64
			var ok bool

			if metric == "wind" {
				value, ok = extractWindSpeed(record)
			} else if metric == "hellmann" {
				// Hellmann exponent berechnen
				value, ok = calculateHellmannExponent(record)
			} else {
				ok = false
			}

			if ok && !record.Time.IsZero() && isFiniteNumber(value) {
				points = append(points, TimeSeriesPoint{
					Time:  record.Time.UTC().Format(time.RFC3339Nano),
					Value: value,
				})
			}
		}
	}

	return points
}

func calculateHellmannExponent(record models.WindRecord) (float64, bool) {
	v10m, ok := extractWindSpeed(record)
	if !ok || v10m <= 0 {
		return 0.0, false
	}

	v100m := extractWindSpeedAtHeight(record, 100)
	if v100m <= 0 {
		return 0.0, false
	}

	// Hellmann-Exponent: alpha = ln(v2/v1) / ln(h2/h1)
	alpha := math.Log(v100m/v10m) / math.Log(100.0/10.0)

	// Prüfen ob Alpha im plausiblen Bereich
	if math.IsNaN(alpha) || alpha <= 0 || alpha >= 1.0 {
		return 0.0, false
	}

	return alpha, true
}

func extractWindSpeedAtHeight(record models.WindRecord, height int) float64 {
	var ptr *float64
	switch height {
	case 10:
		ptr = record.WindData.WindSpeed_10m
	case 80:
		ptr = record.WindData.WindSpeed_80m
	case 100:
		ptr = record.WindData.WindSpeed_100m
	case 120:
		ptr = record.WindData.WindSpeed_120m
	case 180:
		ptr = record.WindData.WindSpeed_180m
	case 200:
		ptr = record.WindData.WindSpeed_200m
	default:
		return 0.0
	}

	if ptr == nil || math.IsNaN(*ptr) || math.IsInf(*ptr, 0) || *ptr < 0 {
		return 0.0
	}
	return *ptr
}

func extractWindSpeed(record models.WindRecord) (float64, bool) {
	for _, candidate := range []*float64{
		record.WindData.WindSpeed_10m,
		record.WindData.WindSpeed_80m,
		record.WindData.WindSpeed_100m,
	} {
		if candidate != nil && !math.IsNaN(*candidate) && !math.IsInf(*candidate, 0) && *candidate >= 0 {
			return *candidate, true
		}
	}
	return 0.0, false
}

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
