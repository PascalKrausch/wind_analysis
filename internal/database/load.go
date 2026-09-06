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
	SELECT time, location_name,
	       wind_speed_10m, wind_speed_80m, wind_speed_100m, wind_speed_120m, wind_speed_180m, wind_speed_200m,
	       wind_direction_10m, wind_direction_80m, wind_direction_100m, wind_direction_120m, wind_direction_180m, wind_direction_200m
	FROM wind_logs
	WHERE location_name = $1 AND time >= $2 AND time <= $3
	ORDER BY time;`

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

		err := rows.Scan(&record.Time, &record.Location.Name,
			&windSpeed10m, &windSpeed80m, &windSpeed100m, &windSpeed120m, &windSpeed180m, &windSpeed200m,
			&windDirection10m, &windDirection80m, &windDirection100m, &windDirection120m, &windDirection180m, &windDirection200m)
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

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fehler beim Iterieren der Zeilen: %w", err)
	}

	return records, nil
}
