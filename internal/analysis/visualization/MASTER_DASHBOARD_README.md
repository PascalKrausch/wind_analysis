# Master Dashboard Datenstrukturen

Diese Datei enthält die Datenstrukturen und Mapping-Funktionen für das geplante Master-Dashboard, das alle Analysedaten für mehrere Standorte in einer einzigen HTML-Datei zusammenfasst.

## Hauptstruktur: MasterDashboardData

Die `MasterDashboardData` Struktur enthält alle Analysedaten für mehrere Standorte:

```go
type MasterDashboardData struct {
    // Metadaten
    Locations      []string
    StartTime      time.Time
    EndTime        time.Time

    // Windgeschwindigkeits-Daten (zeitliche Verläufe)
    WindSpeedData  map[string][]TimeSeriesPoint // key: locationName

    // Hellmann-Exponenten (zeitliche Verläufe)
    HellmannData   map[string][]TimeSeriesPoint // key: locationName

    // Verteilungs-Daten (für verschiedene Höhen und Modelle)
    DistributionData map[string][]DistributionPlotInput // key: locationName

    // Validierungs-Metriken
    ValidationData map[string][]MetricRow // key: locationName

    // Error-Shape-Daten (für verschiedene Höhen)
    ErrorShapeData map[string]map[float64]interpolation.ValidationResult // key: locationName
}
```

## Verwendung

### 1. Neues Dashboard erstellen

```go
master := visualization.NewMasterDashboardData()
```

### 2. Daten für einzelne Standorte hinzufügen

```go
// Windgeschwindigkeits-Daten
windPoints := []visualization.TimeSeriesPoint{
    {Time: time.Now(), Value: 5.5},
    {Time: time.Now().Add(time.Hour), Value: 6.0},
}
master.AddWindSpeedData("location1", windPoints)

// Hellmann-Exponenten
hellmannPoints := []visualization.TimeSeriesPoint{
    {Time: time.Now(), Value: 0.15},
}
master.AddHellmannData("location1", hellmannPoints)

// Verteilungs-Daten
distributionInputs := []visualization.DistributionPlotInput{
    {SeriesName: "weibull", LocationName: "location1", HeightM: 100},
}
master.AddDistributionData("location1", distributionInputs)

// Validierungs-Daten
validationRows := []visualization.MetricRow{
    {Location: "location1", HeightM: 100.0, MAE: 0.5, RMSE: 0.7},
}
master.AddValidationData("location1", validationRows)
```

### 3. Komplette Standort-Analysen mappen

Die bequemste Methode ist die Verwendung der Mapping-Funktionen, die existierende Daten aus dem validation processor Format konvertieren:

```go
// Einzelner Standort
master, err := visualization.MapLocationToMasterDashboard(
    location,           // models.Location
    records,            // []models.WindRecord
    exponents,          // []interpolation.HellmannExponentResult
    validationByLocation, // map[string]map[float64]interpolation.ValidationResult
    distributionInputs, // []visualization.DistributionPlotInput
)

// Mehrere Standorte
master, err := visualization.MapMultipleLocationsToMasterDashboard(
    locations,          // []models.Location
    recordsMap,         // map[string][]models.WindRecord
    exponentsMap,      // map[string][]interpolation.HellmannExponentResult
    validationByLocation, // map[string]map[float64]interpolation.ValidationResult
    distributionInputsMap, // map[string][]visualization.DistributionPlotInput
)
```

Beim Mapping werden Validierungsmetriken standortweise übernommen; Werte anderer Standorte werden nicht in die jeweilige Standortansicht gemischt. Die Standortliste enthält auch konfigurierte Standorte ohne Messwerte. Zeitbereiche ignorieren Zeitstempel ohne Wert, und ungültige Windgeschwindigkeiten bzw. Hellmann-Exponenten werden nicht als Datenpunkte übernommen.

### 4. Daten abrufen

```go
// Prüfen ob Daten vorhanden
if master.HasWindSpeedData("location1") {
    // Zeitreihe abrufen
    windTS := master.GetWindSpeedTimeSeries("location1")
    
    // Alle Windgeschwindigkeits-Zeitreihen
    allWindTS := master.GetAllWindSpeedTimeSeries()
}

// Validierungs-Daten
if master.HasValidationData("location1") {
    rows := master.GetValidationRows("location1")
    allRows := master.GetAllValidationRows()
}

// Standortliste
locations := master.GetLocationsForChart()
```

## Master-Dashboard erzeugen und bedienen

`PlotMasterDashboard` rendert Zeitreihen, Validierungsmetriken und Verteilungs-Fits gemeinsam in eine HTML-Datei:

```go
master, err := visualization.MapMultipleLocationsToMasterDashboard(
    locations,
    recordsMap,
    exponentsMap,
    validationByLocation,
    distributionInputsMap,
)
if err != nil {
    return err
}
if err := visualization.PlotMasterDashboard(master, outputPath); err != nil {
    return err
}
```

Das Dashboard bietet Standortauswahl mit Mehrfachauswahl für Vergleiche. Der Zeitraumfilter gilt ausschließlich für Windgeschwindigkeits- und Hellmann-Zeitreihen; Validierungsmetriken und Verteilungs-Fits werden davon nicht verändert. Fehlen für einen ausgewählten Standort im gefilterten Zeitraum darstellbare Werte, erscheint beim betreffenden Zeitreihen-Chart eine Fehlermeldung statt einer leeren Grafik.

Der Analyse-Lauf für den Standortvergleich speichert das Dashboard als `master_dashboard.html` im konfigurierten Ausgabeverzeichnis. Bestehende Einzel- und Vergleichsausgaben werden weiterhin erzeugt.