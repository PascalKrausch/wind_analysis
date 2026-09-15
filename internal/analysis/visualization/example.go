package visualization

import (
	"fmt"
	"time"
)

// ExampleUsage zeigt ein einfaches Beispiel für die Verwendung der Visualization-Funktionen
func ExampleUsage() {
	// Chart-Konfiguration erstellen
	config := ChartConfig{
		Title:      "Hellmann Exponent über Zeit",
		Subtitle:   "Alpha-Werte für verschiedene Standorte",
		XAxisName:  "Zeit",
		YAxisName:  "Alpha",
		Width:      "1200px",
		Height:     "600px",
		Theme:      "westeros",
	}

	// Line-Chart erstellen
	line := CreateLineChart(config)

	// Beispieldaten
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 2, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 3, 12, 0, 0, 0, time.UTC),
	}
	alphaValues := []float64{0.15, 0.18, 0.16}

	// Zeitachsen-Daten konvertieren
	converter := NewTimeSeriesConverter("2006-01-02")
	xAxis := converter.ConvertTimesToXAxis(times)

	// Daten zum Chart hinzufügen
	AddLineSeries(line, "Standort A", xAxis, alphaValues)

	// Chart in Datei rendern
	filename := "/tmp/hellmann_example.html"
	err := RenderToFile(line, filename)
	if err != nil {
		fmt.Printf("Fehler beim Rendern: %v\n", err)
		return
	}

	fmt.Printf("Chart erfolgreich gerendert: %s\n", filename)
}

// ExampleTimeSeriesConversion zeigt Zeitreihen-Konvertierung
func ExampleTimeSeriesConversion() {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 2, 15, 30, 0, 0, time.UTC),
		time.Date(2022, 1, 3, 8, 45, 0, 0, time.UTC),
	}

	// Verschiedene Zeitformate
	daily := ConvertTimesToXAxisDaily(times)
	hourly := ConvertTimesToXAxisHourly(times)
	monthly := ConvertTimesToXAxisMonthly(times)

	fmt.Printf("Täglich: %v\n", daily)
	fmt.Printf("Stündlich: %v\n", hourly)
	fmt.Printf("Monatlich: %v\n", monthly)
}

// ExampleDataAnalysis zeigt Datenanalyse-Funktionen
func ExampleDataAnalysis() {
	times := []time.Time{
		time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 1, 15, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 2, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 2, 15, 12, 0, 0, 0, time.UTC),
	}
	values := []float64{0.15, 0.18, 0.16, 0.20}

	// Monatliche Durchschnittswerte
	monthlyAvg, counts := CalculateMonthlyAverage(times, values)

	fmt.Println("Monatliche Durchschnittswerte:")
	for month, avg := range monthlyAvg {
		fmt.Printf("%s: %.3f (n=%d)\n", month, avg, counts[month])
	}

	// Statistiken
	min, max, mean, stdDev := CalculateStatistics(values)
	fmt.Printf("Statistiken: Min=%.3f, Max=%.3f, Mean=%.3f, StdDev=%.3f\n", min, max, mean, stdDev)
}

// ExampleScatterPlot zeigt ein Scatter-Plot Beispiel
func ExampleScatterPlot() {
	config := ChartConfig{
		Title:      "Vorhergesagte vs. Tatsächliche Werte",
		Subtitle:   "Modellvalidierung für 80m Höhe",
		XAxisName:  "Tatsächlich",
		YAxisName:  "Vorhergesagt",
		Width:      "800px",
		Height:     "600px",
		Theme:      "westeros",
	}

	scatter := CreateScatterChart(config)

	// Beispieldaten
	actual := []float64{3.0, 4.0, 5.0, 6.0, 7.0}
	predicted := []float64{3.1, 3.9, 5.2, 5.8, 7.1}

	// Scatter-Daten konvertieren
	scatterData := ConvertToScatterData(actual, predicted)

	// Daten zum Chart hinzufügen
	AddScatterSeries(scatter, "Modellvorhersage", scatterData)

	// Chart rendern
	filename := "/tmp/scatter_example.html"
	err := RenderToFile(scatter, filename)
	if err != nil {
		fmt.Printf("Fehler beim Rendern: %v\n", err)
		return
	}

	fmt.Printf("Scatter-Plot erfolgreich gerendert: %s\n", filename)
}
