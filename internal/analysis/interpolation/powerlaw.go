package interpolation

// In powerlaw.go wird die Windgeschwindigkeit in verschiedenen Höhen interpoliert. Die Interpolation erfolgt nach dem Power-Law-Modell, das die Windgeschwindigkeit in Abhängigkeit von der Höhe beschreibt. Die Formel lautet:
// v(h) = v_ref * (h / h_ref) ^ alpha
// Dabei ist:
// - v(h): Windgeschwindigkeit in Höhe h
// - v_ref: Referenz-Windgeschwindigkeit in Höhe h_ref
// - h: Zielhöhe, für die die Windgeschwindigkeit berechnet werden soll
// - h_ref: Referenzhöhe, für die die Windgeschwindigkeit bekannt ist
// - alpha: Exponent, der die Windgeschwindigkeitsänderung mit der Höhe beschreibt (typischerweise zwischen 0.1 und 0.4)

// Es soll zunächst der Hellmann Exponent zu jedem Zeitpunkt und Ort für die Höhen 10m und 100m berechnet werden.
// Dann mit den anderen verfügbaren Höhen (ab 2022) (80m, 120m, 180m, 200m) das erhaltene Modell validiert werden.

import (
	"math"
)

// CalculateHellmannExponent berechnet den Hellmann-Exponent (alpha) basierend auf zwei bekannten Windgeschwindigkeiten und deren Höhen.
func CalculateHellmannExponent(v1, v2 float64, h1, h2 float64) float64 {
	if v1 <= 0 || v2 <= 0 || h1 <= 0 || h2 <= 0 {
		return math.NaN() // Ungültige Eingaben
	}
	return math.Log(v2/v1) / math.Log(h2/h1)
}

// InterpolateWindSpeed berechnet die Windgeschwindigkeit in einer Zielhöhe basierend auf einer bekannten Windgeschwindigkeit und dem Hellmann-Exponent.
func InterpolateWindSpeed(v_ref, h_ref, h_target, alpha float64) float64 {
	if v_ref <= 0 || h_ref <= 0 || h_target <= 0 {
		return math.NaN() // Ungültige Eingaben
	}
	return v_ref * math.Pow(h_target/h_ref, alpha)
}
