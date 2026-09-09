package database

import (
	"context"
	"fmt"
	"wind_analysis/models"

	"github.com/jackc/pgx/v5"
)

func (db *DB) SaveLocation(ctx context.Context, location models.Location) error {
	// Überprüfen, ob der Ort bereits existiert
	var exists bool
	err := db.Conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM locations WHERE location_name=$1)", location.Name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("fehler beim Überprüfen des Ortes: %w", err)
	}

	if !exists {
		// Wenn der Ort nicht existiert, füge ihn hinzu
		_, err = db.Conn.Exec(ctx, "INSERT INTO locations (location_name, latitude, longitude) VALUES ($1, $2, $3)", location.Name, location.Latitude, location.Longitude)
		if err != nil {
			return fmt.Errorf("fehler beim Einfügen des Ortes: %w", err)
		}
	}

	return nil
}

func (db *DB) SaveWindData(ctx context.Context, windRecord models.WindRecord) error {
	// Zuerst sicherstellen, dass der Ort in der Datenbank existiert
	err := db.SaveLocation(ctx, windRecord.Location)
	if err != nil {
		return fmt.Errorf("fehler beim Speichern des Ortes: %w", err)
	}

	// Dann die Winddaten speichern
	_, err = db.Conn.Exec(ctx, `INSERT INTO wind_logs (time, location_name,
		wind_speed_10m, wind_speed_80m, wind_speed_100m, wind_speed_120m, wind_speed_180m, wind_speed_200m,
		wind_direction_10m, wind_direction_80m, wind_direction_100m, wind_direction_120m, wind_direction_180m, wind_direction_200m)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		windRecord.Time, windRecord.Location.Name,
		windRecord.WindData.WindSpeed_10m, windRecord.WindData.WindSpeed_80m, windRecord.WindData.WindSpeed_100m,
		windRecord.WindData.WindSpeed_120m, windRecord.WindData.WindSpeed_180m, windRecord.WindData.WindSpeed_200m,
		windRecord.WindData.WindDirection_10m, windRecord.WindData.WindDirection_80m, windRecord.WindData.WindDirection_100m,
		windRecord.WindData.WindDirection_120m, windRecord.WindData.WindDirection_180m, windRecord.WindData.WindDirection_200m)
	if err != nil {
		return fmt.Errorf("fehler beim Einfügen der Winddaten: %w", err)
	}

	return nil
}

func (db *DB) SaveBatchWindData(ctx context.Context, records []models.WindRecord) error {
	tx, err := db.Conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("fehler beim Starten der Transaktion: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1) Orte deduplizieren und speichern
	seenLocations := make(map[string]models.Location)
	for _, record := range records {
		seenLocations[record.Location.Name] = record.Location
	}

	for _, location := range seenLocations {
		_, err := tx.Exec(ctx, `
            INSERT INTO locations (location_name, latitude, longitude)
            VALUES ($1, $2, $3)
            ON CONFLICT (location_name) DO UPDATE
            SET latitude = EXCLUDED.latitude,
                longitude = EXCLUDED.longitude
        `, location.Name, location.Latitude, location.Longitude)
		if err != nil {
			return fmt.Errorf("fehler beim Speichern des Ortes: %w", err)
		}
	}

	// 2) wind_logs per CopyFrom einfügen
	rows := make([][]any, 0, len(records))
	for _, record := range records {
		rows = append(rows, []any{
			record.Time,
			record.Location.Name,
			record.WindData.WindSpeed_10m,
			record.WindData.WindSpeed_80m,
			record.WindData.WindSpeed_100m,
			record.WindData.WindSpeed_120m,
			record.WindData.WindSpeed_180m,
			record.WindData.WindSpeed_200m,
			record.WindData.WindDirection_10m,
			record.WindData.WindDirection_80m,
			record.WindData.WindDirection_100m,
			record.WindData.WindDirection_120m,
			record.WindData.WindDirection_180m,
			record.WindData.WindDirection_200m,
		})
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"wind_logs"},
		[]string{
			"time", "location_name",
			"wind_speed_10m", "wind_speed_80m", "wind_speed_100m", "wind_speed_120m", "wind_speed_180m", "wind_speed_200m",
			"wind_direction_10m", "wind_direction_80m", "wind_direction_100m", "wind_direction_120m", "wind_direction_180m", "wind_direction_200m",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("fehler beim CopyFrom der Winddaten: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("fehler beim Commit der Transaktion: %w", err)
	}
	return nil
}
