package statistics

import (
	"math"
	"math/cmplx" // Paket für komplexe Zahlen

	"gonum.org/v1/gonum/dsp/fourier"
)

type FrequencyComponent struct {
	Frequency float64
	Amplitude float64
	Phase     float64
}

type FFTResult struct {
	Components []FrequencyComponent
}

// FFT führt eine Fast Fourier Transformation durch
// sampleInterval: Der zeitliche Abstand zwischen Werten (z.B. 1.0 für stündlich)
func FFT(values []float64, sampleInterval float64) FFTResult {
	N := len(values)
	fft := fourier.NewFFT(N)

	// coefficients ist vom Typ []complex128
	coefficients := fft.Coefficients(nil, values)

	components := make([]FrequencyComponent, len(coefficients))

	for i, c := range coefficients {
		// 1. Frequenz des aktuellen Bins berechnen
		freq := float64(i) / (float64(N) * sampleInterval)

		// 2. Amplitude berechnen (Betrag der komplexen Zahl)
		// Wichtig für physikalische Werte: Wir müssen durch N normieren.
		// Ab Bin 1 (i > 0) verdoppeln wir die Amplitude, da die Energie
		// auf positive und negative Frequenzen aufgeteilt ist.
		amp := cmplx.Abs(c) / float64(N)
		if i > 0 {
			amp *= 2
		}

		// 3. Phase berechnen (in Radiant: -Pi bis Pi)
		phase := cmplx.Phase(c)

		components[i] = FrequencyComponent{
			Frequency: freq,
			Amplitude: amp,
			Phase:     phase,
		}
	}

	return FFTResult{Components: components}
}

type PowerComponent struct {
	Frequency float64 // Frequenz (z. B. in 1/Stunde)
	Power     float64 // Leistung (Mean Squared Amplitude, z. B. °C² oder MW²)
}

type PowerSpectrumResult struct {
	Components []PowerComponent
}

// PowerSpectrum berechnet das einseitige Leistungsspektrum (One-Sided Power Spectrum)
// direkt auf Basis des zuvor berechneten FFTResults.
func PowerSpectrum(fft FFTResult) PowerSpectrumResult {
	powerComponents := make([]PowerComponent, len(fft.Components))

	for i, comp := range fft.Components {
		var p float64
		if i == 0 {
			// DC-Komponente (Gleichwert): P = A²
			p = comp.Amplitude * comp.Amplitude
		} else {
			// Für Wechselgrößen (i > 0): P = A² / 2
			p = (comp.Amplitude * comp.Amplitude) / 2.0
		}

		powerComponents[i] = PowerComponent{
			Frequency: comp.Frequency,
			Power:     p,
		}
	}

	return PowerSpectrumResult{Components: powerComponents}
}

// PowerSpectralDensity (PSD) berechnet die Leistungsdichte pro Frequenzeinheit (z. B. MW² / Hz).
// Dies ist wichtig, wenn Signale mit unterschiedlicher Abtastrate verglichen werden sollen.
func PowerSpectralDensity(fft FFTResult, sampleInterval float64) PowerSpectrumResult {
	if len(fft.Components) == 0 {
		return PowerSpectrumResult{}
	}

	// Gesamtdauer der Zeitreihe T = N * dt
	// Die Frequenzauflösung deltaF ist 1 / T
	N := (len(fft.Components) - 1) * 2 // Rekonstruktion der ursprünglichen Zeitreihenlänge N
	if N <= 0 {
		N = 1
	}
	deltaF := 1.0 / (float64(N) * sampleInterval)

	psdResult := PowerSpectrum(fft)

	// Leistung durch Frequenzbandbreite deltaF teilen
	for i := range psdResult.Components {
		if deltaF > 0 {
			psdResult.Components[i].Power /= deltaF
		}
	}

	return psdResult
}

// HannWindow erzeugt ein Hann-Fenster (Hanning) der Länge L.
// Es schwächt die Ränder des Segments auf 0 ab, um Artefakte an den Grenzen zu vermeiden.
func HannWindow(L int) []float64 {
	w := make([]float64, L)
	if L == 1 {
		w[0] = 1.0
		return w
	}
	for n := 0; n < L; n++ {
		w[n] = 0.5 * (1.0 - math.Cos(2.0*math.Pi*float64(n)/float64(L-1)))
	}
	return w
}

// WelchPSD berechnet die Leistungsspektraldichte (PSD) nach der Welch-Methode.
//
// Paramter:
//   - values: Die vollständige Eingabe-Zeitreihe (z. B. SMARD-Residuallast)
//   - segmentLength: Länge eines einzelnen Fensters L (z. B. 512 oder 1024 Stunden).
//     Ein größeres L gibt höhere Frequenzauflösung, ein kleineres L glättet strammer.
//   - overlapFraction: Überlappung der Segmente zwischen 0.0 und 0.9 (Standard: 0.5 für 50%)
//   - sampleInterval: Zeitlicher Abstand zwischen Messwerten (z. B. 1.0 für stündliche Daten)
func WelchPSD(values []float64, segmentLength int, overlapFraction float64, sampleInterval float64) PowerSpectrumResult {
	N := len(values)
	if N < segmentLength || segmentLength <= 0 {
		return PowerSpectrumResult{}
	}

	// 1. Schrittweite (Shift) berechnen basierend auf der Überlappung
	step := int(float64(segmentLength) * (1.0 - overlapFraction))
	if step < 1 {
		step = 1
	}

	// 2. Hann-Fenster generieren und Fenstenergie U zur Skalierung berechnen
	window := HannWindow(segmentLength)
	var winPowerSum float64
	for _, w := range window {
		winPowerSum += w * w
	}
	U := winPowerSum / float64(segmentLength) // Normierungsfaktor U

	// 3. Anzahl der verfügbaren Segmente ermitteln
	numSegments := 0
	for start := 0; start+segmentLength <= N; start += step {
		numSegments++
	}

	if numSegments == 0 {
		return PowerSpectrumResult{}
	}

	// Setup der FFT für die Segmentlänge L
	fft := fourier.NewFFT(segmentLength)
	numCoeffs := segmentLength/2 + 1
	accumulatedPower := make([]float64, numCoeffs)
	segmentBuf := make([]float64, segmentLength)

	// 4. Schleife über alle überlappenden Segmente
	for start := 0; start+segmentLength <= N; start += step {
		// Segment rausschneiden und mit Fenster w[n] gewichten
		for i := 0; i < segmentLength; i++ {
			segmentBuf[i] = values[start+i] * window[i]
		}

		// FFT für dieses gefensterte Segment ausführen
		coeffs := fft.Coefficients(nil, segmentBuf)

		// Periodogramm des Segments berechnen und aufsummieren
		for i, c := range coeffs {
			mag := cmplx.Abs(c)
			// Skalierung für Periodogramm mit Fenster: |X_w(f)|^2 / (L * U)
			p := (mag * mag) / (float64(segmentLength) * U)

			// Einseitiges Spektrum: Verdopplung aller Frequenzen außer DC (i=0) und Nyquist
			if i > 0 && i < len(coeffs)-1 {
				p *= 2.0
			}

			accumulatedPower[i] += p
		}
	}

	// 5. Mittelwert über alle Segmente bilden und in PSD-Einheiten (Power/Hz bzw. Power/(1/h)) umrechnen
	deltaF := 1.0 / (float64(segmentLength) * sampleInterval)
	components := make([]PowerComponent, numCoeffs)

	for i := 0; i < numCoeffs; i++ {
		freq := float64(i) * deltaF
		avgPSD := (accumulatedPower[i] / float64(numSegments)) / deltaF

		components[i] = PowerComponent{
			Frequency: freq,
			Power:     avgPSD,
		}
	}

	return PowerSpectrumResult{Components: components}
}
