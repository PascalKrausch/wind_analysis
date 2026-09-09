package pipeline

import (
	"fmt"
	"math"
	"time"

	"wind_analysis/models"
)

// Layout-Format für das Parsen von Datums-Strings aus der Config
const dateFormat = "2006-01-02"

// GenerateLocationTasks erzeugt eine Liste von FetchTasks basierend auf der Standort-Liste und Zeit-Konfiguration.
func GenerateLocationTasks(cfg *models.Config) ([]models.FetchTask, error) {
	// Datumsangaben aus der Config parsen
	startTime, err := time.Parse(dateFormat, cfg.Timeframe.Start)
	if err != nil {
		return nil, fmt.Errorf("ungültiges Startdatum '%s': %w", cfg.Timeframe.Start, err)
	}

	endTime, err := time.Parse(dateFormat, cfg.Timeframe.End)
	if err != nil {
		return nil, fmt.Errorf("ungültiges Enddatum '%s': %w", cfg.Timeframe.End, err)
	}

	if startTime.After(endTime) {
		return nil, fmt.Errorf("startdatum (%s) liegt nach Enddatum (%s)", cfg.Timeframe.Start, cfg.Timeframe.End)
	}

	// Falls ChunkYears <= 0 oder größer als der Zeitraum ist, fordern wir alles auf einmal an
	chunkYears := cfg.Timeframe.ChunkYears
	totalYears := int(endTime.Sub(startTime).Hours() / 24 / 365)
	if chunkYears <= 0 || chunkYears > totalYears {
		chunkYears = totalYears + 1
	}

	var tasks []models.FetchTask

	// Über die Standort-Liste iterieren
	for _, location := range cfg.LocationList {
		// Koordinaten auf 4 Nachkommastellen runden zur Vermeidung von IEEE-754 Ungenauigkeiten
		roundedLat := roundTo4Decimals(location.Latitude)
		roundedLon := roundTo4Decimals(location.Longitude)

		// Zeitraum in überschaubare Jahresscheiben (Chunks) zerlegen
		currentStart := startTime
		for currentStart.Before(endTime) {
			currentEnd := currentStart.AddDate(chunkYears, 0, 0)
			if currentEnd.After(endTime) {
				currentEnd = endTime
			}

			// Eindeutige Task-ID generieren für Caching / Failure Tracking
			taskID := fmt.Sprintf("%s_%s_%s",
				location.Name,
				currentStart.Format("20060102"),
				currentEnd.Format("20060102"),
			)

			tasks = append(tasks, models.FetchTask{
				ID:           taskID,
				LocationName: location.Name,
				Latitude:     roundedLat,
				Longitude:    roundedLon,
				StartDate:    currentStart,
				EndDate:      currentEnd,
				RetryCount:   0,
			})

			// Weiter zum nächsten Zeitabschnitt
			currentStart = currentEnd
		}
	}

	return tasks, nil
}

// Helper zum Runden von Koordinaten, um Floating-Point-Artefakte zu verhindern
func roundTo4Decimals(val float64) float64 {
	return math.Round(val*10000) / 10000
}
