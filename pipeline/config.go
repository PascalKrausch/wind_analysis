package pipeline

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"wind_analysis/models"
)

// Load liest die YAML-Konfigurationsdatei ein und wendet ggf. Environment-Overrides an.
func Load(path string) (*models.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("konfigurationsdatei konnte nicht geöffnet werden (%s): %w", path, err)
	}
	defer file.Close()

	var cfg models.Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("fehler beim Decodieren der YAML-Datei: %w", err)
	}

	// Validierung der Kernparameter
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("ungültige Konfiguration: %w", err)
	}

	return &cfg, nil
}

// validate prüft kritische Einstellungen vor dem Start der Pipeline
func validate(cfg *models.Config) error {
	if len(cfg.LocationList) == 0 {
		return fmt.Errorf("locationlist darf nicht leer sein")
	}
	if cfg.Pipeline.Concurrency <= 0 {
		cfg.Pipeline.Concurrency = 1
	}
	if cfg.Pipeline.RateLimitRPS <= 0 {
		cfg.Pipeline.RateLimitRPS = 10 // Sicherer Default für Open-Meteo
	}
	return nil
}
