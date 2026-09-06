package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
	"wind_analysis/internal/client"
	"wind_analysis/internal/database"
	"wind_analysis/models"

	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// Kommandozeilen-Argumente
	locationName := flag.String("location", "Berlin", "Name des Ortes")
	lat := flag.Float64("lat", 52.52, "Breitengrad")
	lon := flag.Float64("lon", 13.405, "Längengrad")
	startDate := flag.String("start", "2024-01-01", "Startdatum (YYYY-MM-DD)")
	endDate := flag.String("end", "2024-01-02", "Enddatum (YYYY-MM-DD)")
	flag.Parse()

	// 1) DB initialisieren
	err := godotenv.Load()
	if err != nil {
		log.Println("keine .env gefunden")
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		user,
		password,
		host,
		port,
		dbname,
	)

	db, err := database.New(ctx, connString)
	if err != nil {
		log.Fatalf("DB-Verbindung fehlgeschlagen: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("Schema-Init fehlgeschlagen: %v", err)
	}

	// 2) Winddaten von Open-Meteo API holen
	apiClient := client.NewOpenMeteoClient()

	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Ungültiges Startdatum: %v", err)
	}

	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		log.Fatalf("Ungültiges Enddatum: %v", err)
	}

	fmt.Printf("🌪️  Hole Winddaten für %s (%.4f, %.4f) von %s bis %s...\n",
		*locationName, *lat, *lon, *startDate, *endDate)

	response, err := apiClient.FetchWindDataWithFallback(*lat, *lon, start, end)
	if err != nil {
		log.Fatalf("Fehler beim Abrufen der Winddaten: %v", err)
	}

	// 3) Verfügbare Höhen validieren
	availableHeights, err := client.ValidateWindData(response)
	if err != nil {
		log.Fatalf("Fehler beim Validieren der Winddaten: %v", err)
	}

	fmt.Printf("✅ Verfügbare Höhen: %v\n", availableHeights)
	fmt.Printf("📊 Anzahl der Datenpunkte: %d\n", len(response.Hourly.Time))

	// 4) Daten in Datenbank-Format konvertieren
	locationData, err := client.ConvertToLocationData(response, *locationName)
	if err != nil {
		log.Fatalf("Fehler beim Konvertieren der Daten: %v", err)
	}

	// 5) Location speichern
	location := models.Location{
		Name:      *locationName,
		Latitude:  *lat,
		Longitude: *lon,
	}

	err = db.SaveLocation(ctx, location)
	if err != nil {
		log.Fatalf("Fehler beim Speichern des Ortes: %v", err)
	}

	// 6) Winddaten als Batch speichern
	records := make([]models.WindRecord, 0, len(locationData.Times))

	for i, t := range locationData.Times {
		windData := models.WindData{}

		// Windgeschwindigkeiten
		if val, ok := locationData.Parameters["wind_speed_10m"]; ok && i < len(val) {
			windData.WindSpeed_10m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_speed_80m"]; ok && i < len(val) {
			windData.WindSpeed_80m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_speed_100m"]; ok && i < len(val) {
			windData.WindSpeed_100m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_speed_120m"]; ok && i < len(val) {
			windData.WindSpeed_120m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_speed_180m"]; ok && i < len(val) {
			windData.WindSpeed_180m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_speed_200m"]; ok && i < len(val) {
			windData.WindSpeed_200m = &val[i]
		}

		// Windrichtungen
		if val, ok := locationData.Parameters["wind_direction_10m"]; ok && i < len(val) {
			windData.WindDirection_10m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_direction_80m"]; ok && i < len(val) {
			windData.WindDirection_80m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_direction_100m"]; ok && i < len(val) {
			windData.WindDirection_100m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_direction_120m"]; ok && i < len(val) {
			windData.WindDirection_120m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_direction_180m"]; ok && i < len(val) {
			windData.WindDirection_180m = &val[i]
		}
		if val, ok := locationData.Parameters["wind_direction_200m"]; ok && i < len(val) {
			windData.WindDirection_200m = &val[i]
		}

		record := models.WindRecord{
			Time:     t,
			Location: location,
			WindData: windData,
		}
		records = append(records, record)
	}

	// 7) Batch-Insert in Datenbank
	err = db.SaveBatchWindData(ctx, records)
	if err != nil {
		log.Fatalf("Fehler beim Speichern der Winddaten: %v", err)
	}

	fmt.Printf("✅ Erfolgreich %d Winddatensätze für %s in der Datenbank gespeichert!\n", len(records), *locationName)
}
