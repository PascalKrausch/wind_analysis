package database

import (
	"context"
	"fmt"
	"wind_analysis/models"
)

func SafeVal[T ~int | ~int32 | ~int64 | ~float32 | ~float64](p *T) float64 {
	if p == nil {
		return 0.0
	}
	return float64(*p)
}

func (db *DB) LoadWindData(ctx context.Context, locationName string, startTime, endTime string) ([]models.WindRecord, error) {
	query := `
	SELECT w.time, w.location_name,
	       w.wind_speed_10m, w.wind_speed_80m, w.wind_speed_100m, w.wind_speed_120m, w.wind_speed_180m, w.wind_speed_200m,
	       w.wind_direction_10m, w.wind_direction_80m, w.wind_direction_100m, w.wind_direction_120m, w.wind_direction_180m, w.wind_direction_200m,
	       l.grid_latitude, l.grid_longitude, l.grid_elevation
	FROM wind_logs w
	JOIN locations l ON w.location_name = l.location_name
	WHERE w.location_name = $1 AND w.time >= $2 AND w.time <= $3
	ORDER BY w.time;`

	rows, err := db.Conn.Query(ctx, query, locationName, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Abfragen der Winddaten: %w", err)
	}
	defer rows.Close()

	var records []models.WindRecord

	for rows.Next() {
		var record models.WindRecord
		var windSpeed10m, windSpeed80m, windSpeed100m, windSpeed120m, windSpeed180m, windSpeed200m *float64
		var windDirection10m, windDirection80m, windDirection100m, windDirection120m, windDirection180m, windDirection200m *float64
		var gridLatitude, gridLongitude, gridElevation *float64

		err := rows.Scan(&record.Time, &record.Location.Name,
			&windSpeed10m, &windSpeed80m, &windSpeed100m, &windSpeed120m, &windSpeed180m, &windSpeed200m,
			&windDirection10m, &windDirection80m, &windDirection100m, &windDirection120m, &windDirection180m, &windDirection200m,
			&gridLatitude, &gridLongitude, &gridElevation)
		if err != nil {
			return nil, fmt.Errorf("fehler beim Scannen der Zeile: %w", err)
		}

		record.WindData.WindSpeed_10m = windSpeed10m
		record.WindData.WindSpeed_80m = windSpeed80m
		record.WindData.WindSpeed_100m = windSpeed100m
		record.WindData.WindSpeed_120m = windSpeed120m
		record.WindData.WindSpeed_180m = windSpeed180m
		record.WindData.WindSpeed_200m = windSpeed200m

		record.WindData.WindDirection_10m = windDirection10m
		record.WindData.WindDirection_80m = windDirection80m
		record.WindData.WindDirection_100m = windDirection100m
		record.WindData.WindDirection_120m = windDirection120m
		record.WindData.WindDirection_180m = windDirection180m
		record.WindData.WindDirection_200m = windDirection200m

		if gridLatitude != nil && gridLongitude != nil && gridElevation != nil {
			record.GridMetadata = models.GridMetadata{
				GridLatitude:  *gridLatitude,
				GridLongitude: *gridLongitude,
				GridElevation: *gridElevation,
			}
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fehler beim Iterieren der Zeilen: %w", err)
	}

	return records, nil
}
