package statistics

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/stat"
)

// AdditiveDecompositionResult hält die Ergebnisse einer additiven Zeitreihen-Dekomposition: Y(t) = Trend(t) + Seasonal(t) + Residual(t)
type AdditiveDecompositionResult struct {
	Trend     []float64 `json:"trend"`     // Gleitender Trend T(t)
	Seasonal  []float64 `json:"seasonal"`  // Wiederkehrendes saisonales Muster S(t)
	Residuals []float64 `json:"residuals"` // Verbleibendes Rauschen/Residuum e(t)
}

// MultiplicativeDecompositionResult hält die Ergebnisse einer multiplikativen Dekomposition: Y(t) = Trend(t) * Seasonal(t) * Residual(t)
type MultiplicativeDecompositionResult struct {
	Trend     []float64 `json:"trend"`     // Gleitender Trend T(t)
	Seasonal  []float64 `json:"seasonal"`  // Saisonale Skalierungsfaktoren S(t) (um 1.0 zentriert)
	Residuals []float64 `json:"residuals"` // Verbleibende Skalierungsfehler e(t)
}

// LMDISingleStepResult beschreibt die log-mittelwertbasierte Zerlegung (LMDI) zwischen zwei Zeitpunkten t0 und tT
type LMDISingleStepResult struct {
	DeltaTotal        float64 `json:"delta_total"`        // Exakte Gesamtänderung (E_T - E_0)
	DeltaCapacity     float64 `json:"delta_capacity"`     // Zubau-Effekt (Beitrag der Nennleistung)
	DeltaEfficiency   float64 `json:"delta_efficiency"`   // Auslastungs-/Wetter-Effekt (Beitrag des Kapazitätsfaktors)
	PercentCapacity   float64 `json:"percent_capacity"`   // Prozentualer Anteil des Zubaus am Gesamteffekt
	PercentEfficiency float64 `json:"percent_efficiency"` // Prozentualer Anteil des Wetter/Effizienz-Effekts
}

// calcCenteredMovingAverage berechnet den zentrierten gleitenden Mittelwert für eine Serie
func calcCenteredMovingAverage(series []float64, period int) []float64 {
	n := len(series)
	trend := make([]float64, n)
	halfWindow := period / 2

	for i := 0; i < n; i++ {
		if i < halfWindow || i >= n-halfWindow {
			trend[i] = math.NaN()
			continue
		}

		if period%2 == 0 {
			left := series[i-halfWindow]
			right := series[i+halfWindow]
			inner := series[i-halfWindow+1 : i+halfWindow] // len = period-1
			weightedSum := 0.5*left + stat.Mean(inner, nil)*float64(len(inner)) + 0.5*right
			trend[i] = weightedSum / float64(period)
		} else {
			window := series[i-halfWindow : i+halfWindow+1]
			trend[i] = stat.Mean(window, nil)
		}
	}

	return trend
}

func meanByCounts(values []float64, counts []int) float64 {
	tmp := make([]float64, 0, len(values))
	for i, v := range values {
		if counts[i] > 0 {
			tmp = append(tmp, v)
		}
	}
	if len(tmp) == 0 {
		return 0
	}
	return stat.Mean(tmp, nil)
}

// ClassicalDecompositionAdditive führt eine klassische additive Dekomposition mittels zentriertem gleitendem Mittelwert durch.
// period bestimmt die Zykluslänge (z. B. 24 für Stundendaten mit Tagesgang, 168 für Wochengang).
func ClassicalDecompositionAdditive(series []float64, period int) (AdditiveDecompositionResult, error) {
	n := len(series)
	if period < 2 || n < 2*period {
		return AdditiveDecompositionResult{}, errors.New("series length must be at least twice the period size")
	}

	trend := calcCenteredMovingAverage(series, period)

	// 2. Detrending (Y - T) & Saisonalitätsmittelwert pro Periode berechnen
	seasonalPattern := make([]float64, period)
	seasonalCounts := make([]int, period)

	for i := 0; i < n; i++ {
		if !math.IsNaN(trend[i]) {
			idx := i % period
			seasonalPattern[idx] += series[i] - trend[i]
			seasonalCounts[idx]++
		}
	}

	for p := 0; p < period; p++ {
		if seasonalCounts[p] > 0 {
			seasonalPattern[p] /= float64(seasonalCounts[p])
		}
	}

	// Zentrierung des Saisonsignals (Summe über einen vollen Zyklus muss exakt 0 sein)
	meanSeasonal := meanByCounts(seasonalPattern, seasonalCounts)
	for p := 0; p < period; p++ {
		seasonalPattern[p] -= meanSeasonal
	}

	// 3. Rekonstruktion & Residuen-Berechnung: e(t) = Y(t) - T(t) - S(t)
	seasonal := make([]float64, n)
	residuals := make([]float64, n)

	for i := 0; i < n; i++ {
		seasonal[i] = seasonalPattern[i%period]
		if !math.IsNaN(trend[i]) {
			residuals[i] = series[i] - trend[i] - seasonal[i]
		} else {
			residuals[i] = math.NaN()
		}
	}

	return AdditiveDecompositionResult{
		Trend:     trend,
		Seasonal:  seasonal,
		Residuals: residuals,
	}, nil
}

// ClassicalDecompositionMultiplicative führt eine klassische multiplikative Dekomposition durch: Y(t) = T(t) * S(t) * e(t)
// Geeignet für Signale, deren Saisonschwankungen proportional zur Gesamthöhe wachsen.
func ClassicalDecompositionMultiplicative(series []float64, period int) (MultiplicativeDecompositionResult, error) {
	n := len(series)
	if period < 2 || n < 2*period {
		return MultiplicativeDecompositionResult{}, errors.New("series length must be at least twice the period size")
	}

	// Prüfung auf strikt positive Werte (Multiplikatives Modell benötigt Y > 0)
	for _, v := range series {
		if v <= 0 {
			return MultiplicativeDecompositionResult{}, errors.New("multiplicative decomposition requires strictly positive values")
		}
	}

	trend := calcCenteredMovingAverage(series, period)

	// 2. Detrending durch Division (Y / T)
	seasonalPattern := make([]float64, period)
	seasonalCounts := make([]int, period)

	for i := 0; i < n; i++ {
		if !math.IsNaN(trend[i]) {
			idx := i % period
			seasonalPattern[idx] += series[i] / trend[i]
			seasonalCounts[idx]++
		}
	}

	for p := 0; p < period; p++ {
		if seasonalCounts[p] > 0 {
			seasonalPattern[p] /= float64(seasonalCounts[p])
		}
	}

	// Normalisierung: Das mittlere Saisonverhältnis muss genau 1.0 ergeben
	meanSeasonal := meanByCounts(seasonalPattern, seasonalCounts)
	if meanSeasonal > 0 {
		for p := 0; p < period; p++ {
			seasonalPattern[p] /= meanSeasonal
		}
	}

	// 3. Rekonstruktion & Residuen: e(t) = Y(t) / (T(t) * S(t))
	seasonal := make([]float64, n)
	residuals := make([]float64, n)

	for i := 0; i < n; i++ {
		seasonal[i] = seasonalPattern[i%period]
		if !math.IsNaN(trend[i]) && (trend[i]*seasonal[i]) != 0 {
			residuals[i] = series[i] / (trend[i] * seasonal[i])
		} else {
			residuals[i] = math.NaN()
		}
	}

	return MultiplicativeDecompositionResult{
		Trend:     trend,
		Seasonal:  seasonal,
		Residuals: residuals,
	}, nil
}

// CalcLMDIFirstOrder zerlegt die Differenz zweier Volumenwerte (z. B. Jahreserzeugung E) exakt
// in Zubau-Effekt (Änderung der Leistung P) und Effizienz-/Wetter-Effekt (Änderung des Kapazitätsfaktors CF).
// Basiert auf dem Logarithmic Mean Divisia Index (LMDI I) ohne verbleibendes Residuum.
func CalcLMDIFirstOrder(energy0, energyT, capacity0, capacityT float64) (LMDISingleStepResult, error) {
	if capacity0 <= 0 || capacityT <= 0 {
		return LMDISingleStepResult{}, errors.New("capacities must be strictly greater than zero")
	}

	cf0 := energy0 / capacity0
	cfT := energyT / capacityT
	deltaTotal := energyT - energy0

	// Sonderfall: Keinerlei Ändeung im Zielwert
	if deltaTotal == 0 {
		return LMDISingleStepResult{
			DeltaTotal:        0,
			DeltaCapacity:     0,
			DeltaEfficiency:   0,
			PercentCapacity:   0,
			PercentEfficiency: 0,
		}, nil
	}

	// Logarithmisches Mittel L(y0, yT) = (yT - y0) / ln(yT / y0)
	var L float64
	if energy0 == energyT {
		L = energy0
	} else if energy0 <= 0 || energyT <= 0 {
		// Vermeidung negativer oder unendlicher Logarithmen bei Nullerzeugung
		L = (energy0 + energyT) / 2.0
	} else {
		L = (energyT - energy0) / (math.Log(energyT) - math.Log(energy0))
	}

	// Berechnung der ungewichteten Effekte
	deltaCapacity := L * math.Log(capacityT/capacity0)
	deltaEfficiency := L * math.Log(cfT/cf0)

	percentCap := (deltaCapacity / deltaTotal) * 100.0
	percentEff := (deltaEfficiency / deltaTotal) * 100.0

	return LMDISingleStepResult{
		DeltaTotal:        deltaTotal,
		DeltaCapacity:     deltaCapacity,
		DeltaEfficiency:   deltaEfficiency,
		PercentCapacity:   percentCap,
		PercentEfficiency: percentEff,
	}, nil
}
