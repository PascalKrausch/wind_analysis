package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Conn *pgxpool.Pool
}

func New(ctx context.Context, connString string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Parsen des Connection-Strings: %w", err)
	}

	// Performance-Tuning für High-Throughput Pipelines
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("fehler beim Erstellen des Conn-Pools: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("datenbank nicht erreichbar: %w", err)
	}

	return &DB{Conn: pool}, nil
}

func (db *DB) Close() {
	db.Conn.Close()
}

// InitSchema erstellt alle notwendigen Tabellen und konvertiert wind_logs in eine Hypertable
func (db *DB) InitSchema(ctx context.Context) error {
	// 1. Stammdaten-Tabelle für Städte erstellen
	createLocationTable := `
	CREATE TABLE IF NOT EXISTS locations (
		id SERIAL,
		location_name VARCHAR(100) PRIMARY KEY,
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL
	);`

	_, err := db.Conn.Exec(ctx, createLocationTable)
	if err != nil {
		return fmt.Errorf("fehler beim Erstellen der locations-Tabelle: %w", err)
	}

	// 2. TimescaleDB-Zeitreihentabelle mit allen Parametern erstellen
	createWeatherLogsTable := `
	CREATE TABLE IF NOT EXISTS wind_logs (
		time TIMESTAMPTZ NOT NULL,
		location_name VARCHAR(100) NOT NULL REFERENCES locations(location_name),
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
		wind_direction_200m REAL
	);`

	_, err = db.Conn.Exec(ctx, createWeatherLogsTable)
	if err != nil {
		return fmt.Errorf("fehler beim Erstellen der weather_logs-Tabelle: %w", err)
	}

	// 3. weather_logs in eine TimescaleDB-Hypertable umwandeln (falls noch nicht geschehen)
	// Wir partitionieren standardmäßig nach der Zeitspalte 'time'.
	var hypertableExists bool
	checkHypertable := `SELECT EXISTS (SELECT 1 FROM timescaledb_information.hypertables WHERE hypertable_name = 'wind_logs');`
	err = db.Conn.QueryRow(ctx, checkHypertable).Scan(&hypertableExists)

	if err == nil && !hypertableExists {
		_, err = db.Conn.Exec(ctx, "SELECT create_hypertable('wind_logs', 'time');")
		if err != nil {
			return fmt.Errorf("fehler beim Konvertieren in eine Hypertable: %w", err)
		}
		fmt.Println("📊 Tabelle 'wind_logs' erfolgreich in TimescaleDB-Hypertable konvertiert.")
	}

	// 4. Indizes für schnellere Abfragen
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_wind_logs_time ON wind_logs (time DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_wind_logs_location_time ON wind_logs (location_name, time DESC);`,
	}

	for _, idx := range indexes {
		if _, err := db.Conn.Exec(ctx, idx); err != nil {
			return fmt.Errorf("fehler beim Erstellen eines Indexes: %w", err)
		}
	}

	fmt.Println("✅ Datenbank-Schema erfolgreich initialisiert.")
	return nil
}
