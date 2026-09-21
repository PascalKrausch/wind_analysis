package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"wind_analysis/models"
)

// APIEndpointType definiert die verschiedenen API-Endpunkte von Open-Meteo.
type APIEndpointType int

const (
	EndpointArchive APIEndpointType = iota
	EndpointHistoricalForecast
	EndpointForecast
)

// FetchChunk repräsentiert einen zusammenhängenden Zeitraum, der von einem bestimmten API-Endpunkt abgerufen werden soll.
type FetchChunk struct {
	Endpoint APIEndpointType
	Start    time.Time
	End      time.Time
}

// PlanStrategy analysiert das Zeitfenster und zerlegt es in optimale Chunks
func PlanStrategy(start, end time.Time) []FetchChunk {
	now := time.Now().UTC()

	// Begrenze das Enddatum auf heute, da APIs keine zukünftigen Daten liefern
	if end.After(now) {
		end = now
	}

	// Verschiedene Cut-offs basierend auf API-Verfügbarkeit
	historicalForecastCutoff := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC) // Fix auf 2022
	archiveCutoff := now.AddDate(0, 0, -5)                                  // 5 Tage Lag für Archive API

	var chunks []FetchChunk
	currStart := start

	// 1. Archive API (älteste Daten bis ca. 2022, nur 10m/100m)
	if currStart.Before(historicalForecastCutoff) {
		chunkEnd := end
		if chunkEnd.After(historicalForecastCutoff) {
			chunkEnd = historicalForecastCutoff
		}
		chunks = append(chunks, FetchChunk{
			Endpoint: EndpointArchive,
			Start:    currStart,
			End:      chunkEnd,
		})
		currStart = chunkEnd.Add(1 * time.Hour)
	}

	// 2. Historical Forecast API (2022 bis vor 5 Tagen, alle Höhen)
	if currStart.Before(archiveCutoff) && currStart.Before(end) {
		chunkEnd := end
		if chunkEnd.After(archiveCutoff) {
			chunkEnd = archiveCutoff
		}
		chunks = append(chunks, FetchChunk{
			Endpoint: EndpointHistoricalForecast,
			Start:    currStart,
			End:      chunkEnd,
		})
		currStart = chunkEnd.Add(1 * time.Hour)
	}

	// 3. Forecast API (letzte 5 Tage, alle Höhen)
	if !currStart.After(end) {
		chunks = append(chunks, FetchChunk{
			Endpoint: EndpointForecast,
			Start:    currStart,
			End:      end,
		})
	}

	return chunks
}

// SmartClient ist ein intelligenter API-Client, der automatisch die beste Strategie für die Datenabfrage wählt.
type SmartClient struct {
	archiveBaseURL            string
	historicalForecastBaseURL string
	forecastBaseURL           string
}

// NewSmartClient erstellt einen neuen SmartClient mit den Standard-URLs für die Open-Meteo API.
func NewSmartClient() *SmartClient {
	return &SmartClient{
		archiveBaseURL:            "https://archive-api.open-meteo.com/v1/archive",
		historicalForecastBaseURL: "https://historical-forecast-api.open-meteo.com/v1/forecast",
		forecastBaseURL:           "https://api.open-meteo.com/v1/forecast",
	}
}

// FetchSeamlessData ruft Winddaten für einen gegebenen Standort und Zeitraum ab, indem es die beste API-Strategie wählt.
func (c *SmartClient) FetchSeamlessData(ctx context.Context, lat, lon float64, start, end time.Time) (*models.OpenMeteoResponse, error) {
	chunks := PlanStrategy(start, end)

	// Sequentiell statt parallel
	results := make([]*models.OpenMeteoResponse, len(chunks))
	for i, chunk := range chunks {
		resp, err := c.executeFetch(ctx, chunk, lat, lon)
		if err != nil {
			return nil, fmt.Errorf("fehler in Chunk %d: %w", i, err)
		}
		results[i] = resp
	}

	return MergeResponses(results), nil
}

// executeFetch führt die eigentliche API-Abfrage für einen gegebenen Chunk aus.
func (c *SmartClient) executeFetch(ctx context.Context, chunk FetchChunk, lat, lon float64) (*models.OpenMeteoResponse, error) {
	var baseURL string
	var hourlyParams string

	switch chunk.Endpoint {
	case EndpointArchive:
		baseURL = c.archiveBaseURL
		// Archive API unterstützt nur 10m und 100m
		hourlyParams = "wind_speed_10m,wind_speed_100m,wind_direction_10m,wind_direction_100m"
	case EndpointHistoricalForecast:
		baseURL = c.historicalForecastBaseURL
		// Historical Forecast API unterstützt alle Höhen
		hourlyParams = "wind_speed_10m,wind_speed_80m,wind_speed_100m,wind_speed_120m,wind_speed_180m,wind_speed_200m,wind_direction_10m,wind_direction_80m,wind_direction_100m,wind_direction_120m,wind_direction_180m,wind_direction_200m"
	case EndpointForecast:
		baseURL = c.forecastBaseURL
		// Forecast API unterstützt alle Höhen und verwendet past_days statt start/end_date
		hourlyParams = "wind_speed_10m,wind_speed_80m,wind_speed_100m,wind_speed_120m,wind_speed_180m,wind_speed_200m,wind_direction_10m,wind_direction_80m,wind_direction_100m,wind_direction_120m,wind_direction_180m,wind_direction_200m"
	}

	// URL-Params bauen (inkl. dynamischer Höhenparameter)
	params := url.Values{}
	params.Add("latitude", fmt.Sprintf("%.4f", lat))
	params.Add("longitude", fmt.Sprintf("%.4f", lon))
	params.Add("hourly", hourlyParams)
	params.Add("wind_speed_unit", "ms") // m/s statt Standard (km/h)

	// Verschiedene Datums-Parameter je nach API
	if chunk.Endpoint == EndpointForecast {
		// Forecast API verwendet past_days für historische Daten
		pastDays := int(time.Since(chunk.Start).Hours() / 24)
		if pastDays < 0 {
			pastDays = 0
		}
		params.Add("past_days", fmt.Sprintf("%d", pastDays))
	} else {
		// Archive und Historical Forecast API verwenden start_date/end_date
		params.Add("start_date", chunk.Start.Format("2006-01-02"))
		params.Add("end_date", chunk.End.Format("2006-01-02"))
	}

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// HTTP-Request ausführen (hier vereinfacht dargestellt)
	return c.doHTTPRequest(ctx, reqURL)
}

// doHTTPRequest führt den HTTP-Request aus und parst die Antwort in ein OpenMeteoResponse-Objekt.
func (c *SmartClient) doHTTPRequest(ctx context.Context, url string) (*models.OpenMeteoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fehler beim erstellen des requests: %w", err)
	}

	client := &http.Client{
		Timeout: 120 * time.Second, // Erhöhter Timeout für Historical Forecast API
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fehler beim ausführen des HTTP requests: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fehler beim lesen der antwort: %w", err)
	}

	var response models.OpenMeteoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("fehler beim JSON decoding: %w", err)
	}

	log.Printf("Empfangene Einheit für Wind: %s", response.HourlyUnits.WindSpeed10m)

	return &response, nil
}

// MergeResponses führt gegliederte OpenMeteoResponses zeitlich sortiert zusammen
func MergeResponses(responses []*models.OpenMeteoResponse) *models.OpenMeteoResponse {
	if len(responses) == 0 {
		return nil
	}
	if len(responses) == 1 {
		return responses[0]
	}

	// Initialisiere merged response mit Daten der ersten response
	merged := &models.OpenMeteoResponse{
		Latitude:  responses[0].Latitude,
		Longitude: responses[0].Longitude,
		Elevation: responses[0].Elevation,
		Timezone:  responses[0].Timezone,
		Hourly:    responses[0].Hourly,
	}

	// Füge alle weiteren responses hinzu
	for i := 1; i < len(responses); i++ {
		resp := responses[i]

		// Überprüfe Konsistenz der Metadaten
		if resp.Latitude != merged.Latitude || resp.Longitude != merged.Longitude {
			continue // Überspringe inkonsistente responses
		}

		// Ermittle, ob der erste Zeitstempel von resp mit dem letzten Zeitstempel von merged
		// übereinstimmt (Chunk-Grenzen können sich überlappen). Der Skip-Count wird anhand des
		// Time-Arrays bestimmt und dann konsistent auf ALLE parallelen Value-Slices angewendet,
		// damit Zeit- und Messwert-Arrays niemals gegeneinander verschoben werden.
		skip := 0
		if len(merged.Hourly.Time) > 0 && len(resp.Hourly.Time) > 0 &&
			resp.Hourly.Time[0] == merged.Hourly.Time[len(merged.Hourly.Time)-1] {
			skip = 1
		}

		merged.Hourly.Time = appendUniqueStrings(merged.Hourly.Time, resp.Hourly.Time, skip)

		// Füge Windgeschwindigkeiten hinzu
		merged.Hourly.WindSpeed_10m = appendFloatPointers(merged.Hourly.WindSpeed_10m, resp.Hourly.WindSpeed_10m, skip)
		merged.Hourly.WindSpeed_80m = appendFloatPointers(merged.Hourly.WindSpeed_80m, resp.Hourly.WindSpeed_80m, skip)
		merged.Hourly.WindSpeed_100m = appendFloatPointers(merged.Hourly.WindSpeed_100m, resp.Hourly.WindSpeed_100m, skip)
		merged.Hourly.WindSpeed_120m = appendFloatPointers(merged.Hourly.WindSpeed_120m, resp.Hourly.WindSpeed_120m, skip)
		merged.Hourly.WindSpeed_180m = appendFloatPointers(merged.Hourly.WindSpeed_180m, resp.Hourly.WindSpeed_180m, skip)
		merged.Hourly.WindSpeed_200m = appendFloatPointers(merged.Hourly.WindSpeed_200m, resp.Hourly.WindSpeed_200m, skip)

		// Füge Windrichtungen hinzu
		merged.Hourly.WindDirection_10m = appendFloatPointers(merged.Hourly.WindDirection_10m, resp.Hourly.WindDirection_10m, skip)
		merged.Hourly.WindDirection_80m = appendFloatPointers(merged.Hourly.WindDirection_80m, resp.Hourly.WindDirection_80m, skip)
		merged.Hourly.WindDirection_100m = appendFloatPointers(merged.Hourly.WindDirection_100m, resp.Hourly.WindDirection_100m, skip)
		merged.Hourly.WindDirection_120m = appendFloatPointers(merged.Hourly.WindDirection_120m, resp.Hourly.WindDirection_120m, skip)
		merged.Hourly.WindDirection_180m = appendFloatPointers(merged.Hourly.WindDirection_180m, resp.Hourly.WindDirection_180m, skip)
		merged.Hourly.WindDirection_200m = appendFloatPointers(merged.Hourly.WindDirection_200m, resp.Hourly.WindDirection_200m, skip)
	}

	return merged
}

// appendUniqueStrings hängt new an base an und überspringt dabei genau `skip` Elemente
// am Anfang von new (Chunk-Grenzen-Überlappung). Der Skip-Count wird zentral in
// MergeResponses anhand des Time-Arrays ermittelt, damit alle parallelen Arrays
// (Zeit, Windgeschwindigkeit, Windrichtung) exakt im Gleichschritt bleiben.
func appendUniqueStrings(base, new []string, skip int) []string {
	if skip > len(new) {
		skip = len(new)
	}
	return append(base, new[skip:]...)
}

// appendFloatPointers hängt new an base an und überspringt dabei genau `skip` Elemente
// am Anfang von new. Werte bleiben *float64, damit fehlende Messwerte (nil/JSON-null)
// nicht mit echten 0.0-Werten verwechselt werden.
func appendFloatPointers(base, new []*float64, skip int) []*float64 {
	if skip > len(new) {
		skip = len(new)
	}
	return append(base, new[skip:]...)
}

// transformResponseToRecords mappt die parallelen JSON-Arrays der API auf flache Zeilenstrukturen.
func (c *SmartClient) TransformResponseToRecords(task models.FetchTask, resp *models.OpenMeteoResponse) ([]models.WindRecord, error) {
	totalHours := len(resp.Hourly.Time)
	if totalHours == 0 {
		return nil, nil
	}

	records := make([]models.WindRecord, totalHours)
	loc := models.Location{
		Name:      task.LocationName,
		Latitude:  task.Latitude,
		Longitude: task.Longitude,
	}

	// Grid-Metadaten aus der API-Antwort
	gridMetadata := models.GridMetadata{
		GridLatitude:  resp.Latitude,
		GridLongitude: resp.Longitude,
		GridElevation: resp.Elevation,
	}

	for i := 0; i < totalHours; i++ {
		// Parst den ISO-Zeitstempel (z.B. "2020-01-01T00:00")
		t, err := time.Parse("2006-01-02T15:00", resp.Hourly.Time[i])
		if err != nil {
			// Fallback auf flexibles Parsing, falls Iso-Format abweicht
			t, _ = time.Parse(time.RFC3339, resp.Hourly.Time[i])
		}

		records[i] = models.WindRecord{
			Time:     t,
			Location: loc,
			WindData: models.WindData{
				WindSpeed_10m:      getPointerAtIndex(resp.Hourly.WindSpeed_10m, i),
				WindSpeed_80m:      getPointerAtIndex(resp.Hourly.WindSpeed_80m, i),
				WindSpeed_100m:     getPointerAtIndex(resp.Hourly.WindSpeed_100m, i),
				WindSpeed_120m:     getPointerAtIndex(resp.Hourly.WindSpeed_120m, i),
				WindSpeed_180m:     getPointerAtIndex(resp.Hourly.WindSpeed_180m, i),
				WindSpeed_200m:     getPointerAtIndex(resp.Hourly.WindSpeed_200m, i),
				WindDirection_10m:  getPointerAtIndex(resp.Hourly.WindDirection_10m, i),
				WindDirection_80m:  getPointerAtIndex(resp.Hourly.WindDirection_80m, i),
				WindDirection_100m: getPointerAtIndex(resp.Hourly.WindDirection_100m, i),
				WindDirection_120m: getPointerAtIndex(resp.Hourly.WindDirection_120m, i),
				WindDirection_180m: getPointerAtIndex(resp.Hourly.WindDirection_180m, i),
				WindDirection_200m: getPointerAtIndex(resp.Hourly.WindDirection_200m, i),
			},
			GridMetadata: gridMetadata,
		}
	}

	return records, nil
}

// getPointerAtIndex prüft Out-Of-Bounds und gibt den Zeiger an der gegebenen Position zurück.
// Ein nil-Eintrag (JSON-null der API, d.h. fehlender Messwert) bleibt dabei nil und wird
// so korrekt als DB-NULL gespeichert statt als 0.0.
func getPointerAtIndex(slice []*float64, index int) *float64 {
	if index < 0 || index >= len(slice) {
		return nil
	}
	return slice[index]
}
