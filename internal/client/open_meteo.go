package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"wind_analysis/models"
)

const (
	baseURL = "https://archive-api.open-meteo.com/v1/archive"
)

// OpenMeteoClient ist ein Client für die Open-Meteo API
type OpenMeteoClient struct {
	client *http.Client
}

// NewOpenMeteoClient erstellt einen neuen Open-Meteo Client
func NewOpenMeteoClient() *OpenMeteoClient {
	return &OpenMeteoClient{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetAvailableHeightParameters gibt die verfügbaren Höhenparameter basierend auf dem Datum zurück
func getAvailableHeightParameters(date time.Time) string {
	// Für Daten vor 2022: nur 10m und 100m (Historical Forecast API)
	if date.Before(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return "wind_speed_10m,wind_speed_100m,wind_direction_10m,wind_direction_100m"
	}

	// Für Daten ab 2022: volles Profil (Historical Forecast API)
	return "wind_speed_10m,wind_speed_80m,wind_speed_100m,wind_speed_120m,wind_speed_180m,wind_speed_200m," +
		"wind_direction_10m,wind_direction_80m,wind_direction_100m,wind_direction_120m,wind_direction_180m,wind_direction_200m"
}

// buildAPIURL baut die API-URL für einen Request
func buildAPIURL(lat, lon float64, startDate, endDate time.Time) (string, error) {
	params := url.Values{}
	params.Add("latitude", fmt.Sprintf("%.6f", lat))
	params.Add("longitude", fmt.Sprintf("%.6f", lon))
	params.Add("hourly", getAvailableHeightParameters(startDate))
	params.Add("start_date", startDate.Format("2006-01-02"))
	params.Add("end_date", endDate.Format("2006-01-02"))
	params.Add("timezone", "auto")

	return fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil
}

// FetchWindDataWithFallback ruft Winddaten mit Fallback-Strategie ab
func (c *OpenMeteoClient) FetchWindDataWithFallback(lat, lon float64, startDate, endDate time.Time) (*models.OpenMeteoResponse, error) {
	// Versuchen Sie zuerst das volle Profil
	apiURL, err := buildAPIURL(lat, lon, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to build API URL: %w", err)
	}

	response, err := c.fetch(apiURL)
	if err == nil {
		return response, nil
	}

	// Fallback auf reduziertes Profil für ältere Daten
	legacyParams := url.Values{}
	legacyParams.Add("latitude", fmt.Sprintf("%.6f", lat))
	legacyParams.Add("longitude", fmt.Sprintf("%.6f", lon))
	legacyParams.Add("hourly", "wind_speed_10m,wind_speed_100m,wind_direction_10m,wind_direction_100m")
	legacyParams.Add("start_date", startDate.Format("2006-01-02"))
	legacyParams.Add("end_date", endDate.Format("2006-01-02"))
	legacyParams.Add("timezone", "auto")

	legacyURL := fmt.Sprintf("%s?%s", baseURL, legacyParams.Encode())
	return c.fetch(legacyURL)
}

// fetch führt den eigentlichen API-Request aus
func (c *OpenMeteoClient) fetch(apiURL string) (*models.OpenMeteoResponse, error) {
	resp, err := c.client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResponse models.OpenMeteoResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &apiResponse, nil
}

// ValidateWindData prüft, welche Höhen tatsächlich Daten enthalten
func ValidateWindData(response *models.OpenMeteoResponse) ([]string, error) {
	hasData := func(field []float64) bool {
		return len(field) > 0
	}

	availableHeights := []string{}
	if hasData(response.Hourly.WindSpeed_10m) {
		availableHeights = append(availableHeights, "10m")
	}
	if hasData(response.Hourly.WindSpeed_80m) {
		availableHeights = append(availableHeights, "80m")
	}
	if hasData(response.Hourly.WindSpeed_100m) {
		availableHeights = append(availableHeights, "100m")
	}
	if hasData(response.Hourly.WindSpeed_120m) {
		availableHeights = append(availableHeights, "120m")
	}
	if hasData(response.Hourly.WindSpeed_180m) {
		availableHeights = append(availableHeights, "180m")
	}
	if hasData(response.Hourly.WindSpeed_200m) {
		availableHeights = append(availableHeights, "200m")
	}

	if len(availableHeights) == 0 {
		return nil, fmt.Errorf("no wind data available")
	}

	return availableHeights, nil
}

// ConvertToLocationData konvertiert die API-Antwort in LocationData für die Datenbank
func ConvertToLocationData(response *models.OpenMeteoResponse, locationName string) (*models.LocationData, error) {
	if len(response.Hourly.Time) == 0 {
		return nil, fmt.Errorf("no time data in response")
	}

	// Parse times
	times := make([]time.Time, 0, len(response.Hourly.Time))
	for _, timeStr := range response.Hourly.Time {
		t, err := time.Parse("2006-01-02T15:04", timeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse time %s: %w", timeStr, err)
		}
		times = append(times, t)
	}

	// Build parameters map
	parameters := make(map[string][]float64)

	if len(response.Hourly.WindSpeed_10m) > 0 {
		parameters["wind_speed_10m"] = response.Hourly.WindSpeed_10m
	}
	if len(response.Hourly.WindSpeed_80m) > 0 {
		parameters["wind_speed_80m"] = response.Hourly.WindSpeed_80m
	}
	if len(response.Hourly.WindSpeed_100m) > 0 {
		parameters["wind_speed_100m"] = response.Hourly.WindSpeed_100m
	}
	if len(response.Hourly.WindSpeed_120m) > 0 {
		parameters["wind_speed_120m"] = response.Hourly.WindSpeed_120m
	}
	if len(response.Hourly.WindSpeed_180m) > 0 {
		parameters["wind_speed_180m"] = response.Hourly.WindSpeed_180m
	}
	if len(response.Hourly.WindSpeed_200m) > 0 {
		parameters["wind_speed_200m"] = response.Hourly.WindSpeed_200m
	}

	if len(response.Hourly.WindDirection_10m) > 0 {
		parameters["wind_direction_10m"] = response.Hourly.WindDirection_10m
	}
	if len(response.Hourly.WindDirection_80m) > 0 {
		parameters["wind_direction_80m"] = response.Hourly.WindDirection_80m
	}
	if len(response.Hourly.WindDirection_100m) > 0 {
		parameters["wind_direction_100m"] = response.Hourly.WindDirection_100m
	}
	if len(response.Hourly.WindDirection_120m) > 0 {
		parameters["wind_direction_120m"] = response.Hourly.WindDirection_120m
	}
	if len(response.Hourly.WindDirection_180m) > 0 {
		parameters["wind_direction_180m"] = response.Hourly.WindDirection_180m
	}
	if len(response.Hourly.WindDirection_200m) > 0 {
		parameters["wind_direction_200m"] = response.Hourly.WindDirection_200m
	}

	return &models.LocationData{
		Name:       locationName,
		Times:      times,
		Parameters: parameters,
	}, nil
}
