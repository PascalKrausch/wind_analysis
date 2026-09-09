package interpolation

//In vector.go wird die Windrichtung von Grad in die u- und v-Komponente umgerechnet. Die u-Komponente ist die Ost-West-Komponente, die v-Komponente ist die Nord-Süd-Komponente. Die Umrechnung erfolgt nach den Formeln:
// u = wind_speed * sin(wind_direction)
// v = wind_speed * cos(wind_direction)
// Dabei ist wind_direction in Grad angegeben, wobei 0° nach Norden zeigt und im Uhrzeigersinn zunimmt (90° = Osten, 180° = Süden, 270° = Westen).

import (
	"math"
)

const (
	pi = math.Pi
)

// Umrechnung von Grad in Bogenmaß
func degToRad(deg float64) float64 {
	return deg * (pi / 180)
}

// DirectionToVector wandelt die Windrichtung in Bogenmaß in Vektorkomponenten um
func DirectionToVektor(wind_direction float64) (u, v float64) {
	// Windrichtung in Bogenmaß umrechnen
	rad := degToRad(wind_direction)

	// u- und v-Komponenten berechnen
	u = math.Sin(rad)
	v = math.Cos(rad)

	return u, v
}

// VelocityToVector wandelt die Windgeschwindigkeit und -richtung in Vektorkomponenten um
func VelocityToVector(wind_speed, wind_direction float64) (u, v float64) {
	// Windrichtung in Bogenmaß umrechnen
	rad := degToRad(wind_direction)

	// u- und v-Komponenten berechnen
	u = wind_speed * math.Sin(rad)
	v = wind_speed * math.Cos(rad)

	return u, v
}

// RadToDeg wandelt Bogenmaß in Grad um
func RadToDeg(rad float64) float64 {
	return rad * (180 / pi)
}

// VectorToDirection wandelt die u- und v-Komponenten in Windrichtung in Grad um
func VectorToDirection(u, v float64) float64 {
	// Windrichtung in Bogenmaß berechnen
	rad := math.Atan2(u, v)

	// In Grad umrechnen
	deg := RadToDeg(rad)

	// Normalisieren auf [0, 360)
	if deg < 0 {
		deg += 360
	}

	return deg
}

func VectorToVelocity(u, v float64) (wind_speed, wind_direction float64) {
	// Windgeschwindigkeit berechnen
	wind_speed = math.Sqrt(u*u + v*v)

	// Windrichtung berechnen
	wind_direction = VectorToDirection(u, v)

	return wind_speed, wind_direction
}
