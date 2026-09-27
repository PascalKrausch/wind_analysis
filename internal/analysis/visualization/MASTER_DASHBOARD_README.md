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

## Integration in die Analyse-Pipeline

Die Datenstrukturen sind so konzipiert, dass sie sich nahtlos in die existierende Analyse-Pipeline integrieren lassen:

```go
// In cmd/analyser/main.go oder validation/processor.go
func RunMasterDashboardAnalysis(ctx context.Context, db *database.DB, config Config, locationList []models.Location) error {
    master := visualization.NewMasterDashboardData()
    
    for _, location := range locationList {
        // Daten laden wie bisher
        records, err := db.LoadWindData(ctx, location.Name, config.StartTime, config.EndTime)
        if err != nil {
            continue
        }
        
        // Analysen durchführen wie bisher
        exponents := interpolation.CalculateHellmannExponentsForDataset(records)
        validationByLocation := interpolation.ValidatePowerLawModelByLocation(records, exponents)
        
        // Verteilungs-Analyse
        distributionInputs, err := fitting.FindBestFitsForLocation(location.Name, records, fitters, nil)
        
        // Zum Master-Dashboard hinzufügen
        locationMaster, err := visualization.MapLocationToMasterDashboard(
            location, records, exponents, validationByLocation, distributionInputs,
        )
        
        // Daten zusammenführen
        master.WindSpeedData[location.Name] = locationMaster.WindSpeedData[location.Name]
        master.HellmannData[location.Name] = locationMaster.HellmannData[location.Name]
        // ... etc
    }
    
    // Master-Dashboard rendern (nächster Schritt)
    // visualization.PlotMasterDashboard(master, outputPath)
    
    return nil
}
```

## Nächste Schritte

Die implementierten Datenstrukturen und Mapping-Funktionen bilden das Fundament für das Master-Dashboard. Die nächsten Schritte sind:

1. **UI-Komponenten implementieren**: Dropdown für Standortauswahl, Filter-Buttons, Tabs
2. **JavaScript für dynamische Filterung**: Basierend auf dem existierenden `getDynamicZoomJS` Muster
3. **Master-Dashboard Generator**: Die `PlotMasterDashboard` Funktion, die alle Charts in einer Page kombiniert
4. **Integration in die Pipeline**: Anpassung von `cmd/analyser/main.go` um das Master-Dashboard zu generieren

Die aktuelle Implementierung ist vollständig getestet und bereit für die Erweiterung um die UI-Komponenten.