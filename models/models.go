package models

import "time"

// Datenstruktur für die API-Antwort von Open-Meteo
type WindData struct {
	// Windgeschwindigkeiten auf verschiedenen Höhen
	WindSpeed_10m   *float64 `json:"wind_speed_10m"`
	WindSpeed_80m   *float64 `json:"wind_speed_80m"`
	WindSpeed_100m  *float64 `json:"wind_speed_100m"`
	WindSpeed_120m  *float64 `json:"wind_speed_120m"`
	WindSpeed_180m  *float64 `json:"wind_speed_180m"`
	WindSpeed_200m  *float64 `json:"wind_speed_200m"`

	// Windrichtungen auf verschiedenen Höhen
	WindDirection_10m  *float64 `json:"wind_direction_10m"`
	WindDirection_80m  *float64 `json:"wind_direction_80m"`
	WindDirection_100m *float64 `json:"wind_direction_100m"`
	WindDirection_120m *float64 `json:"wind_direction_120m"`
	WindDirection_180m *float64 `json:"wind_direction_180m"`
	WindDirection_200m *float64 `json:"wind_direction_200m"`
}

// Location hält die geografischen Koordinaten eines Ortes
type Location struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Datenstruktur für die Speicherung von Winddaten in der Datenbank
type WindRecord struct {
	Time     time.Time `json:"time"`
	Location Location  `json:"location"`
	WindData WindData  `json:"wind_data"`
}

// LocationData hält die Winddaten für eine bestimmte Location über die Zeit hinweg, um sie in der Datenbank zu speichern
type LocationData struct {
	Name       string               `json:"name"`
	Times      []time.Time          `json:"times"`
	Parameters map[string][]float64 `json:"parameters"`
}

// OpenMeteoResponse repräsentiert die API-Antwort von Open-Meteo
type OpenMeteoResponse struct {
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Timezone  string     `json:"timezone"`
	Hourly    HourlyData `json:"hourly"`
}

// HourlyData enthält die stündlichen Wetterdaten
type HourlyData struct {
	Time                 []string  `json:"time"`
	WindSpeed_10m        []float64 `json:"wind_speed_10m"`
	WindSpeed_80m        []float64 `json:"wind_speed_80m"`
	WindSpeed_100m       []float64 `json:"wind_speed_100m,omitempty"`
	WindSpeed_120m       []float64 `json:"wind_speed_120m"`
	WindSpeed_180m       []float64 `json:"wind_speed_180m"`
	WindSpeed_200m       []float64 `json:"wind_speed_200m,omitempty"`
	WindDirection_10m    []float64 `json:"wind_direction_10m"`
	WindDirection_80m    []float64 `json:"wind_direction_80m"`
	WindDirection_100m   []float64 `json:"wind_direction_100m,omitempty"`
	WindDirection_120m   []float64 `json:"wind_direction_120m"`
	WindDirection_180m   []float64 `json:"wind_direction_180m"`
	WindDirection_200m   []float64 `json:"wind_direction_200m,omitempty"`
}
