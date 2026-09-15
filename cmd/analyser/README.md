# Wind Analyser

Der Wind Analyser führt Power-Law-Interpolation und Visualisierung von Winddaten durch.

## Installation

```bash
go build ./cmd/analyser
```

## Verwendung

### Umgebungsvariablen

Erstellen Sie eine `.env` Datei mit der Datenbankverbindung:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_username
DB_PASSWORD=your_password
DB_NAME=wind_analysis
```

### Einzelne Standortanalyse

Analyse eines einzelnen Standorts mit Power-Law-Interpolation und Visualisierung:

```bash
./analyser -location Husum -start 2022-01-01 -end 2023-12-31 -output ./output
```

**Optionen:**
- `-location`: Name des Standorts (z.B. Husum, Harz, Schwäbische Alb)
- `-compare`: Standortvergleich über alle konfigurierten Standorte
- `-start`: Startzeitpunkt (YYYY-MM-DD, Standard: 2022-01-01)
- `-end`: Endzeitpunkt (YYYY-MM-DD, Standard: heute)
- `-output`: Ausgabeverzeichnis für Charts (Standard: ./output)
- `-validate`: Modellvalidierung durchführen (Standard: true)
- `-validate=false`: Modellvalidierung deaktivieren

**Beispiel:**
```bash
./analyser -location Husum -start 2022-01-01 -end 2023-12-31 -output ./output
```

### Standortvergleich

Vergleich mehrerer Standorte basierend auf der Konfiguration in `config.yaml`:

```bash
./analyser -compare -start 2022-01-01 -end 2023-12-31 -output ./output
```

### Vollständige Analyse (Default)

Ohne Parameter werden alle konfigurierten Standorte analysiert und anschließend ein Standortvergleich erstellt:

```bash
./analyser -start 2022-01-01 -end 2023-12-31 -output ./output
```

## Ausgabe

Der Analyser erstellt folgende Visualisierungen im angegebenen Ausgabeverzeichnis:

### Einzelne Standortanalyse
1. **Timeline**: `hellmann_timeline_<location>.html` - Hellmann-Exponent über Zeit mit DataZoom
2. **Validierungsmetriken**: `validation_metrics_<location>.html` - MAE und RMSE pro Höhe
3. **Scatter-Plots**: `validation_scatter_<height>m_<location>.html` - Vorhergesagte vs. tatsächliche Werte für jede Höhe (80m, 120m, 180m, 200m)

### Standortvergleich
1. **Vergleich**: `location_comparison.html` - Multi-Line-Chart der Hellmann-Exponenten für alle Standorte mit DataZoom

## Chart-Features

Alle generierten Charts verfügen über folgende **responsive und interaktive Features**:

### Responsives Design
- **Automatische Anpassung** an verschiedene Bildschirmgrößen (Desktop, Tablet, Mobile)
- **Optimierte Höhen**: 90vh auf Desktop, 75-85vh auf mobilen Geräten
- **Flexible Breite**: 100% für optimale Darstellung auf allen Geräten

### Interaktive Funktionen
- **Zoom-Funktion**: Vergrößern und Verkleinern von Chart-Bereichen
- **Reset-Button**: Zurücksetzen auf Standardansicht
- **Speichern**: Export als Bild
- **DataZoom-Slider**: Zeitachsen-Navigation bei Timeline-Charts
- **Tooltips**: Detaillierte Informationen bei Mouseover

## Beispiele

### Analyse Husum 2022-2023
```bash
./analyser -location Husum -start 2022-01-01 -end 2023-12-31
```

### Analyse ohne Validierung
```bash
./analyser -location Harz -start 2022-01-01 -end 2023-12-31 -validate=false
```

### Standortvergleich aller konfigurierten Standorte
```bash
./analyser -compare -start 2022-01-01 -end 2023-12-31
```

### Benutzerdefiniertes Ausgabeverzeichnis
```bash
./analyser -location "Schwäbische Alb" -output ./my_charts
```

### Vollständige Analyse aller Standorte
```bash
./analyser -start 2022-01-01 -end 2023-12-31
```

## Power-Law Modell

Der Analyser verwendet das Power-Law-Modell zur Windgeschwindigkeitsinterpolation:

```
v(h) = v_ref * (h / h_ref) ^ alpha
```

Dabei ist:
- `v(h)`: Windgeschwindigkeit in Höhe h
- `v_ref`: Referenz-Windgeschwindigkeit in Höhe h_ref
- `h`: Zielhöhe
- `h_ref`: Referenzhöhe (10m)
- `alpha`: Hellmann-Exponent (berechnet aus 10m und 100m)

## Validierung

Die Modellvalidierung vergleicht die interpolierten Werte mit den tatsächlichen Messwerten für die Höhen 80m, 120m, 180m und 200m (verfügbar ab 2022).

**Metriken:**
- **MAE** (Mean Absolute Error): Durchschnittlicher absoluter Fehler
- **RMSE** (Root Mean Square Error): Wurzel des mittleren quadratischen Fehlers
- **Correlation**: Pearson-Korrelationskoeffizient zwischen vorhergesagten und tatsächlichen Werten

## Implementierungsdetails

### Powerlaw-Visualisierung Funktionen

Die Powerlaw-Visualisierungen werden durch folgende Funktionen im `internal/analysis/visualization/powerlaw_plot.go` Package implementiert:

#### 1. Timeline-Visualisierung
```go
func PlotHellmannExponentTimeline(exponents []interpolation.HellmannExponentResult, outputPath string) error
```
- Zeigt Hellmann-Exponenten über Zeit als Line-Chart
- Aggregiert mehrere Standorte zu Mittelwerten pro Zeitpunkt
- **Features**: DataZoom-Slider für Zeitachsen-Navigation

#### 2. Modellvalidierung Scatter-Plot
```go
func PlotModelValidation(stats interpolation.ValidationStats, height float64, outputPath string) error
```
- Scatter-Plot von vorhergesagten vs. tatsächlichen Werten
- Zeigt Validierungsmetriken im Subtitle (MAE, RMSE, Correlation)
- **Features**: Zoom-Funktion für detaillierte Analyse

#### 3. Validierungsmetriken Bar-Chart
```go
func PlotValidationMetrics(results map[float64]interpolation.ValidationResult, outputPath string) error
```
- Bar-Chart mit MAE und RMSE pro Höhe
- Ermöglicht schnellen Vergleich der Modellgenauigkeit
- **Features**: Zoom und Export-Funktionen

#### 4. Standortvergleich
```go
func PlotMultiLocationComparison(exponents []interpolation.HellmannExponentResult, outputPath string) error
```
- Multi-Line-Chart für mehrere Standorte
- Verbindende Linien für fehlende Datenpunkte
- **Features**: DataZoom-Slider für Zeitachsen-Analyse

### Responsive Chart-Konfiguration

Die Chart-Konfiguration in `internal/analysis/visualization/charts.go` unterstützt:

```go
type ChartConfig struct {
    Title          string
    Subtitle       string
    XAxisName      string
    YAxisName      string
    Width          string
    Height         string
    Theme          string
    Responsive     bool   // Responsive Design aktivieren
    EnableZoom     bool   // Zoom-Funktion aktivieren
    EnableDataZoom bool   // DataZoom für Zeitachsen aktivieren
}
```

**Default-Einstellungen:**
- Width: "1200px", Height: "800px"
- Responsive: true, EnableZoom: true
- Theme: "westeros"

## Interpretation der Ergebnisse

### Hellmann-Exponent
Der Hellmann-Exponent (α) beschreibt wie stark die Windgeschwindigkeit mit der Höhe zunimmt:

- **α < 0.1**: Sehr flaches Profil, starke Durchmischung (typisch tagsüber bei Konvektion)
- **α = 0.1-0.2**: Flaches Profil, instabile Atmosphäre
- **α = 0.2-0.3**: Mittleres Profil, Übergangszustand
- **α > 0.3**: Steiles Profil, stabile Atmosphäre (typisch nachts)

### Tagesgang des Exponenten
Ein typischer Tagesgang zeigt:
- **Morgens (06-10 Uhr)**: Höhere α-Werte (~0.3-0.4) durch stabile Schichtung
- **Mittags (12-16 Uhr)**: Niedrigste α-Werte (~0.1-0.2) durch thermische Konvektion
- **Abends/Nachts**: Mittlere bis hohe α-Werte (~0.25-0.35) bei wieder stabilisierender Atmosphäre

### Validierungsmetriken
- **Korrelation > 0.99**: Exzellente Modellleistung
- **MAE < 1.0 m/s**: Sehr gute Vorhersagegenauigkeit
- **RMSE < 1.5 m/s**: Gute Vorhersagegenauigkeit
- **Korrelation < 0.95**: Modell könnte verbessert werden

## Zu erwartende Periodizitäten

### Meteorologisch begründbare Periodizitäten

**1. Tagesperiodizität (24h)**
- Thermische Konvektion im Tagesgang
- Niedrigere α-Werte tagsüber, höhere nachts
- Maximum der Instabilität um 14 Uhr

**2. Jahresperiodizität (365 Tage)**
- Saisonal-bedingte Änderungen der atmosphärischen Stabilität
- Sommer: Niedrigere α-Werte (mehr Konvektion)
- Winter: Höhere α-Werte (stabilere Schichtung)

**3. Synoptische Periodizitäten (3-7 Tage)**
- Wetterfronten und Tiefdruckgebiete
- Zyklonische Aktivität
- Durchgang von Hoch- und Tiefdrucksystemen

**4. Wöchentliche Periodizität (möglich)**
- Anthropogene Effekte in städtischen Gebieten
- Wochenend-Effekte
- Lokale Wärmeentwicklung

### Analysemethoden für Periodizitäten

Für den Nachweis dieser Periodizitäten könnten folgende Methoden implementiert werden:

- **Fourier-Analyse (FFT)**: Identifikation dominanter Frequenzen
- **Spektralanalyse**: Periodogramm-Darstellung
- **Wavelet-Analyse**: Zeit-Frequenz-Lokalisierung
- **Autokorrelationsanalyse**: Korrelationen über verschiedene Zeitverschiebungen
- **STL-Dekomposition**: Trennung von Trend, Saison und Residuum
