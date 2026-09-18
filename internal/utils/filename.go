package utils

import (
	"path/filepath"
	"strings"
)

// SanitizeForFilename bereinigt einen String für die Verwendung in Dateinamen
func SanitizeForFilename(name string) string {
	s := strings.TrimSpace(strings.ToLower(name))
	s = strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	).Replace(s)
	if s == "" {
		return "unbekannt"
	}
	return s
}

// BuildOutputPath erstellt einen vollständigen Pfad für eine Ausgabedatei
func BuildOutputPath(baseDir, filename string) string {
	return filepath.Join(baseDir, filename)
}