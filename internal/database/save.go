package database

import (
	"context"
	"fmt"
	"time"
	"wind_analysis/models"

	"github.com/jackc/pgx/v5"
)

func (db *DB) SaveLocation(ctx context.Context, location models.Location, gridMetadata models.GridMetadata) error {
	// Überprüfen, ob der Ort bereits existiert
	var exists bool
	err := db.Conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM locations WHERE location_name=$1)", location.Name).Scan(&exists)
	if err != nil {
		return fmt.Errorf("fehler beim Überprüfen des Ortes: %w", err)
	}

	if !exists {
		// Wenn der Ort nicht existiert, füge ihn hinzu mit Grid-Metadaten
		_, err = db.Conn.Exec(ctx, "INSERT INTO locations (location_name, latitude, longitude, grid_latitude, grid_longitude, grid_elevation) VALUES ($1, $2, $3, $4, $5, $6)",
			location.Name, location.Latitude, location.Longitude, gridMetadata.GridLatitude, gridMetadata.GridLongitude, gridMetadata.GridElevation)
		if err != nil {
			return fmt.Errorf("fehler beim Einfügen des Ortes: %w", err)
		}
	} else {
		// Wenn der Ort existiert, aktualisiere die Grid-Metadaten falls nötig
		_, err = db.Conn.Exec(ctx, "UPDATE locations SET grid_latitude = $1, grid_longitude = $2, grid_elevation = $3 WHERE location_name = $4",
			gridMetadata.GridLatitude, gridMetadata.GridLongitude, gridMetadata.GridElevation, location.Name)
		if err != nil {
			return fmt.Errorf("fehler beim Aktualisieren der Grid-Metadaten: %w", err)
		}
	}

	return nil
}

func (db *DB) SaveWindData(ctx context.Context, windRecord models.WindRecord) error {
	// Zuerst sicherstellen, dass der Ort in der Datenbank existiert mit Grid-Metadaten
	err := db.SaveLocation(ctx, windRecord.Location, windRecord.GridMetadata)
	if err != nil {
		return fmt.Errorf("fehler beim Speichern des Ortes: %w", err)
	}

	// Dann die Winddaten speichern (ohne Grid-Metadaten, da diese in locations stehen)
	_, err = db.Conn.Exec(ctx, `INSERT INTO wind_logs (time, location_name,
		wind_speed_10m, wind_speed_80m, wind_speed_100m, wind_speed_120m, wind_speed_180m, wind_speed_200m,
		wind_direction_10m, wind_direction_80m, wind_direction_100m, wind_direction_120m, wind_direction_180m, wind_direction_200m)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (time, location_name) DO UPDATE SET
			wind_speed_10m = EXCLUDED.wind_speed_10m,
			wind_speed_80m = EXCLUDED.wind_speed_80m,
			wind_speed_100m = EXCLUDED.wind_speed_100m,
			wind_speed_120m = EXCLUDED.wind_speed_120m,
			wind_speed_180m = EXCLUDED.wind_speed_180m,
			wind_speed_200m = EXCLUDED.wind_speed_200m,
			wind_direction_10m = EXCLUDED.wind_direction_10m,
			wind_direction_80m = EXCLUDED.wind_direction_80m,
			wind_direction_100m = EXCLUDED.wind_direction_100m,
			wind_direction_120m = EXCLUDED.wind_direction_120m,
			wind_direction_180m = EXCLUDED.wind_direction_180m,
			wind_direction_200m = EXCLUDED.wind_direction_200m`,
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

	if len(records) == 0 {
		return nil
	}

	tx, err := db.Conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("fehler beim Starten der Transaktion: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Orte mit Grid-Metadaten speichern
	seenLocations := make(map[string]struct {
		location     models.Location
		gridMetadata models.GridMetadata
	})

	for _, record := range records {
		seenLocations[record.Location.Name] = struct {
			location     models.Location
			gridMetadata models.GridMetadata
		}{
			location:     record.Location,
			gridMetadata: record.GridMetadata,
		}
	}

	for name, data := range seenLocations {
		_, err := tx.Exec(ctx, `
			INSERT INTO locations (
				location_name,
				latitude,
				longitude,
				grid_latitude,
				grid_longitude,
				grid_elevation
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (location_name)
			DO UPDATE SET
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude,
				grid_latitude = EXCLUDED.grid_latitude,
				grid_longitude = EXCLUDED.grid_longitude,
				grid_elevation = EXCLUDED.grid_elevation
		`,
			name,
			data.location.Latitude,
			data.location.Longitude,
			data.gridMetadata.GridLatitude,
			data.gridMetadata.GridLongitude,
			data.gridMetadata.GridElevation,
		)

		if err != nil {
			return fmt.Errorf("fehler beim Speichern des Ortes: %w", err)
		}
	}

	// 2. Temporäre Staging-Tabelle (ohne Grid-Spalten, da diese in locations stehen)
	_, err = tx.Exec(ctx, `
		CREATE TEMP TABLE wind_logs_stage (
			time TIMESTAMPTZ,
			location_name VARCHAR(100),
			wind_speed_10m REAL,
			wind_speed_80m REAL,
			wind_speed_100m REAL,
			wind_speed_120m REAL,
			wind_speed_180m REAL,
			wind_speed_200m REAL,
			wind_direction_10m REAL,
			wind_direction_80m REAL,
			wind_direction_100m REAL,
			wind_direction_120m REAL,
			wind_direction_180m REAL,
			wind_direction_200m REAL,
			ingest_order BIGINT
		) ON COMMIT DROP
	`)
	if err != nil {
		return fmt.Errorf("fehler beim Erstellen der Staging-Tabelle: %w", err)
	}

	// 3. Daten für CopyFrom vorbereiten (ohne Grid-Metadaten)
	rows := make([][]any, 0, len(records))

	for i, record := range records {
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

			int64(i),
		})
	}

	// 4. Schnell in die Staging-Tabelle kopieren
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"wind_logs_stage"},
		[]string{
			"time",
			"location_name",

			"wind_speed_10m",
			"wind_speed_80m",
			"wind_speed_100m",
			"wind_speed_120m",
			"wind_speed_180m",
			"wind_speed_200m",

			"wind_direction_10m",
			"wind_direction_80m",
			"wind_direction_100m",
			"wind_direction_120m",
			"wind_direction_180m",
			"wind_direction_200m",

			"ingest_order",
		},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return fmt.Errorf("fehler beim CopyFrom in die Staging-Tabelle: %w", err)
	}

	// 5. Deduplizieren + Upsert in die echte Tabelle (ohne Grid-Spalten)
	_, err = tx.Exec(ctx, `
		INSERT INTO wind_logs (
			time,
			location_name,

			wind_speed_10m,
			wind_speed_80m,
			wind_speed_100m,
			wind_speed_120m,
			wind_speed_180m,
			wind_speed_200m,

			wind_direction_10m,
			wind_direction_80m,
			wind_direction_100m,
			wind_direction_120m,
			wind_direction_180m,
			wind_direction_200m
		)
		SELECT
			time,
			location_name,

			wind_speed_10m,
			wind_speed_80m,
			wind_speed_100m,
			wind_speed_120m,
			wind_speed_180m,
			wind_speed_200m,

			wind_direction_10m,
			wind_direction_80m,
			wind_direction_100m,
			wind_direction_120m,
			wind_direction_180m,
			wind_direction_200m
		FROM (
			SELECT
				s.*,
				ROW_NUMBER() OVER (
					PARTITION BY time, location_name
					ORDER BY ingest_order DESC
				) AS rn
			FROM wind_logs_stage s
		) s
		WHERE rn = 1

		ON CONFLICT (time, location_name)
		DO UPDATE SET
			wind_speed_10m = EXCLUDED.wind_speed_10m,
			wind_speed_80m = EXCLUDED.wind_speed_80m,
			wind_speed_100m = EXCLUDED.wind_speed_100m,
			wind_speed_120m = EXCLUDED.wind_speed_120m,
			wind_speed_180m = EXCLUDED.wind_speed_180m,
			wind_speed_200m = EXCLUDED.wind_speed_200m,

			wind_direction_10m = EXCLUDED.wind_direction_10m,
			wind_direction_80m = EXCLUDED.wind_direction_80m,
			wind_direction_100m = EXCLUDED.wind_direction_100m,
			wind_direction_120m = EXCLUDED.wind_direction_120m,
			wind_direction_180m = EXCLUDED.wind_direction_180m,
			wind_direction_200m = EXCLUDED.wind_direction_200m
	`)

	if err != nil {
		return fmt.Errorf("fehler beim Upsert der Winddaten: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("fehler beim Commit der Transaktion: %w", err)
	}

	return nil
}

// ShouldFetchData prüft ob Daten für einen Zeitraum neu geladen werden sollen
// mit NULL-aware Recovery Logik
func (db *DB) ShouldFetchData(ctx context.Context, locationName string, start, end time.Time) (bool, error) {
	var totalCount, nullCount int

	// NULL-aware Query: Prüft Gesamtanzahl und Anzahl von NULL-Werten
	err := db.Conn.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) - COUNT(wind_speed_10m) as null_count
		FROM wind_logs
		WHERE location_name = $1 AND time BETWEEN $2 AND $3
	`, locationName, start, end).Scan(&totalCount, &nullCount)

	if err != nil {
		return true, err // Bei Fehler: fetchen
	}

	// Keine Daten vorhanden → fetchen
	if totalCount == 0 {
		return true, nil
	}

	// NULL-Daten Check (Hauptproblem des Users)
	if nullCount > 0 {
		fmt.Printf("NULL-Daten gefunden (%d/%d) für %s %s-%s → Refetch\n",
			nullCount, totalCount, locationName, start.Format("2006-01-02"), end.Format("2006-01-02"))
		return true, nil
	}

	// Completeness Check (für partielle API-Fehler)
	expectedCount := int(end.Sub(start).Hours())
	if expectedCount > 0 && float64(totalCount) < float64(expectedCount)*0.9 {
		fmt.Printf("Unvollständige Daten (%d/%d erwartet) für %s → Refetch\n",
			totalCount, expectedCount, locationName)
		return true, nil
	}

	return false, nil // Alles gut → skippen
}
