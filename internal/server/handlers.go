package server

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"wind_analysis/models"
)

type LocationResponse struct {
	Name     string  `json:"name"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
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
	HeightM   int      `json:"heightM"`
}

type DistributionResponse struct {
	Location string  `json:"location"`
	HeightM  int     `json:"heightM"`
	Histogram []struct {
		Lower float64 `json:"lower"`
		Upper float64 `json:"upper"`
		Count int     `json:"count"`
	} `json:"histogram"`
}

func (s *Server) handleDistributions(w http.ResponseWriter, r *http.Request) {
	var req DistributionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Placeholder - Verteilungs-Fits sind komplexer und erfordern
	// die Integration mit fitting package. Für jetzt leer.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]DistributionResponse{})
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
