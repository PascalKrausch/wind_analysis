# Wind Analysis - Wind Data Pipeline & Statistical Analysis

**⚠️ Hinweis:** Lernprojekt zur praktischen Anwendung von statistischen Methoden und numerischen Verfahren in Go. Nicht für produktive Windenergie-Planung geeignet.

## 🎯 Projektziel

Statistische Analyse von Windgeschwindigkeitsdaten mittels generischer Verteilungsanalyse, automatischer Modellauswahl und Performance-optimierter Datenverarbeitung.

## 🏗️ Architektur

```
wind-analysis/
├── cmd/
│   ├── fetcher/         # Einfacher Fetcher für einzelne Standorte
│   └── analyser/        # Statistische Analysen & Visualisierungen
├── internal/
│   ├── client/          # Open-Meteo API Client mit Rate-Limiting
│   ├── database/        # PostgreSQL/TimescaleDB Integration
│   ├── utils/           # Utility-Funktionen (Filename-Sanitization, etc.)
│   └── analysis/
│       ├── interpolation/   # Power-Law Windprofil-Interpolation
│       ├── statistics/      # Deskriptive Statistik, Korrelation, Verteilungen
│       ├── distribution/    # Generische Verteilungsanalyse (Weibull, Gamma, Log-Normal)
│       │   ├── types.go            # Interface-Definitionen (ContinuousDistribution, Fitter)
│       │   ├── weibull.go          # Weibull-Verteilung (mit Interface-Implementierung)
│       │   ├── gamma.go            # Gamma-Verteilung
│       │   ├── log_normal.go       # Log-Normal-Verteilung
│       │   ├── fitter.go           # Automatische Modellauswahl (SelectBestModel)
│       │   └── data_cleaning.go    # Generische Datenbereinigung
│       ├── validation/      # Validierungs-Orchestrierung
│       ├── weibull/         # Legacy Weibull-Verarbeitungslogik (wind-spezifisch)
│       └── visualization/   # Chart-Erstellung (go-echarts)
│           ├── distribution_plot.go # Generische Verteilungs-Visualisierung
│           └── weibull_plot.go      # Weibull-spezifische Visualisierung
├── pipeline/            # Concurrency-Engine für parallele Datenabfrage
├── models/              # Datenstrukturen & Konfiguration
├── config.yaml          # Standorte & Pipeline-Konfiguration
└── docker-compose.yml   # Datenbank-Setup
```

## 🏛️ Architektur-Prinzipien

Das Projekt folgt etablierten Software-Architektur-Patterns:

- **Layered Architecture**: Klare Trennung zwischen Entry Points, Business Logic und Data Access
- **Domain-Driven Design**: Packages nach fachlichen Domänen strukturiert
- **Interface-Based Design**: Generische Interfaces für Verteilungen (`ContinuousDistribution`, `Fitter`)
- **Strategy Pattern**: Austauschbare Verteilungs-Implementierungen und Fitting-Algorithmen
- **Dependency Injection**: Konstruktoren nehmen Abhängigkeiten als Parameter für Testbarkeit
- **Context-First Pattern**: Alle asynchronen Funktionen verwenden `context.Context`
- **Pipeline Pattern**: Fan-Out/Fan-In für parallele Datenverarbeitung
- **Repository Pattern**: Datenbankzugriffe gekapselt in dedizierten Packages
- **Separation of Concerns**: Datenbereinigung, Fitting und Visualisierung sind getrennt

## 🔬 Architektur-Details: Generische Verteilungsanalyse

Das Projekt verwendet eine moderne, interface-basierte Architektur für die statistische Analyse:

### Core Interfaces

```go
// ContinuousDistribution beschreibt eine kontinuierliche Wahrscheinlichkeitsverteilung
type ContinuousDistribution interface {
    PDF(x float64) float64
    CDF(x float64) float64
    LogPDF(x float64) float64
    Params() []float64
}

// Fitter beschreibt eine Schnittstelle zur Parameterschätzung
type Fitter interface {
    Fit(data []float64) (ContinuousDistribution, error)
    Name() string
}
```

### Aktuelle Implementierungen

- **Weibull**: `Weibull` struct + `WeibullFitter` (MLE mit Location-Parameter)
- **Gamma**: `Gamma` struct + `GammaFitter` (Momentenmethode + Thom-Näherung)
- **Log-Normal**: `LogNormal` struct + `LogNormalFitter` (Analytisches MLE)

### Automatische Modellauswahl

```go
bestDist, metrics, err := distribution.SelectBestModel(data)
// Wählt automatisch das beste Modell basierend auf AIC
```

### Generische Utilities

- **Datenbereinigung**: `CleanData()`, `CleanDataWithZero()`, `IsFinite()`
- **Statistik**: `Summarize()`, `SortedCopy()`
- **Validierung**: `EvaluateFit()` berechnet AIC, BIC, KS-Test, RMSE
- **Visualisierung**: `BuildDistributionCurves()`, `NewDistributionDashboard()`

### Erweiterbarkeit

Neue Verteilungen können einfach hinzugefügt werden durch:
1. Implementierung von `ContinuousDistribution` Interface
2. Implementierung von `Fitter` Interface
3. Hinzufügen zum `fitters` Array in `SelectBestModel()`

### Rückwärtskompatibilität

Die Architektur behält die Rückwärtskompatibilität zum bestehenden Weibull-spezifischen Code:
- Legacy-Funktionen wie `WeibullPDF()`, `WeibullCDF()` sind weiterhin verfügbar
- `WeibullParams`, `WeibullAnalysisResult` etc. werden für Kompatibilität beibehalten
- Der `weibull/` Processor funktioniert weiterhin mit wind-spezifischen Daten

### Architektur-Refactoring

Der Übergang von Weibull-spezifischer zu generischer Architektur erfolgte in mehreren Schritten:

1. **Interface-Definition**: Einführung von `ContinuousDistribution` und `Fitter` Interfaces
2. **Weibull-Entkopplung**: `Weibull` struct implementiert `ContinuousDistribution`, `WeibullFitter` implementiert `Fitter`
3. **Datenbereinigung**: Extraktion von `cleanWindSpeedData()` zu generischem `CleanData()`
4. **Visualisierung**: Einführung von `DistributionPlotInput` und generischen Chart-Funktionen
5. **Andere Verteilungen**: Gamma und Log-Normal implementierten bereits das generische Muster

### Vorteile der neuen Architektur

- **Erweiterbarkeit**: Neue Verteilungen können ohne Änderung bestehenden Codes hinzugefügt werden
- **Wiederverwendbarkeit**: Generische Funktionen (Datenbereinigung, Validierung, Visualisierung) für alle Verteilungen
- **Testbarkeit**: Interfaces erleichtern Unit-Testing mit Mocks
- **Wartbarkeit**: Klare Trennung der Zuständigkeiten und konsistente APIs
- **Flexibilität**: Automatische Modellauswahl ermöglicht datengetriebene Entscheidungen

## 🚀 Funktionalität

### Implementiert

- **Open-Meteo API Integration**: Abruf von Winddaten für verschiedene Höhen (10m, 80m, 100m, 120m, 180m, 200m)
- **Automatische API-Auswahl**: Intelligente Wahl zwischen Forecast, Historical Forecast und Archive API
- **Concurrency Pipeline**: Parallele Datenabfrage mit konfigurierbaren Worker-Routinen
- **Rate-Limiting**: Konfigurierbare API-Rate-Limits für stabile Datenabfrage
- **Power-Law Interpolation**: Windprofil-Interpolation zwischen verschiedenen Höhen (Hellmann-Exponenten)
- **Batch-Verarbeitung**: Effizientes Speichern großer Datensätze via PostgreSQL COPY
- **Datenbank-Speicherung**: PostgreSQL mit TimescaleDB für effiziente Zeitreihen-Abfragen
- **Generische Verteilungsanalyse**: Interface-basierte Architektur für verschiedene Verteilungen
- **Unterstützte Verteilungen**: Weibull, Gamma, Log-Normal (erweiterbar)
- **Automatische Modellauswahl**: `SelectBestModel()` wählt automatisch das beste Modell basierend auf AIC
- **Parameterschätzung**: MLE-basierte Parameterschätzung für alle Verteilungen
- **Goodness-of-Fit Tests**: KS-Test, AIC, BIC, RMSE für Modellvalidierung
- **Generische Datenbereinigung**: `CleanData()`, `Summarize()`, `SortedCopy()` für alle Verteilungen
- **Generische Visualisierung**: `BuildDistributionCurves()`, `NewDistributionDashboard()` für jede Verteilung
- **Statistische Analyse**: Deskriptive Statistik, Korrelation (Pearson, Spearman, Kendall), Histogramme
- **Visualisierung**: Interaktive HTML-Charts (Histogramme, PDF/CDF, Heatmaps, Vergleiche)
- **Validierung**: Power-Law Modellvalidierung mit Fehleranalyse

### In Planung

- **Erweiterte Verteilungen**: Rayleigh, Normal, Log-Logistic, GEV für Extremwertanalyse
- **Windenergie-Berechnung**: Theoretische Energieerträge und Capacity Factors
- **Statistische Vergleiche**: Wasserstein-Distanz für zeitliche Veränderungen
- **Erweiterte Visualisierungen**: Windrosen, Zeitreihen-Dashboards
- **Processor-Modell-Umbau**: Migration von weibull/ zu generischem distribution/ Processor

## 📋 Voraussetzungen

- Go 1.20+
- PostgreSQL mit TimescaleDB Extension
- Docker & Docker Compose

## 🔧 Installation

```bash
# 1. Repository klonen
git clone <repo-url>
cd wind_analysis

# 2. Abhängigkeiten installieren
go mod download

# 3. Datenbank starten
docker-compose up -d

# 4. Konfiguration anpassen (config.yaml)
# Standorte, Zeitraum und Pipeline-Parameter konfigurieren
```

## 🚀 Nutzung

### Datenabfrage (Pipeline)

Parallele Abfrage mehrerer Standorte für einen Zeitraum:

```bash
# Pipeline starten (lädt Daten für alle Standorte in config.yaml)
go run cmd/fetcher/main.go -config config.yaml
```

Die Pipeline-Konfiguration erfolgt über `config.yaml`:

```yaml
locationlist:
  - name: Husum         <- Ändern oder Ergänzen
    latitude: 54.4878
    longitude: 9.0556
  - name: Harz
    latitude: 51.7967
    longitude: 10.6206

timeframe:              <- Zeitraum anpassen
  start: "2015-01-01"
  end: "2026-09-09"
  chunk_years: 5

pipeline:               <- Go-Worker anpassen
  concurrency: 3      # Parallele Worker-Routinen
  batch_size: 500      # DB Bulk-Insert Schwelle
  rate_limit_rps: 3    # API Rate Limit
```

### API-Endpoint-Logik

Automatische Auswahl basierend auf Zeitraum:
- **Letzte 3 Monate**: Forecast API mit `past_days` (volles Höhenprofil)
- **2022-heute**: Historical Forecast API (volles Höhenprofil)
- **Vor 2022**: Archive API (10m, 100m + Power Law Interpolation)

### Statistische Analyse & Visualisierung

Analyse der geladenen Winddaten mit statistischen Methoden und Visualisierungen:

```bash
# Analyser starten (analysiert Daten für alle Standorte in config.yaml)
go run cmd/analyser/main.go
```

Der Analyser erstellt folgende Ausgaben im `./output` Verzeichnis:

- **Hellmann-Exponenten**: Timeline-Diagramme der Windprofil-Parameter
- **Validierungsmetriken**: Tabellen und Charts für Power-Law Modellvalidierung
- **Weibull-Analyse**: PDF/CDF Charts, Histogramme und Fit-Güte-Metriken
- **Standortvergleich**: Heatmaps und Vergleichs-Charts zwischen verschiedenen Standorten

## 🎓 Lernziele

- **Concurrency Patterns**: Fan-Out/Fan-In Pipeline mit Worker-Pools
- **API-Integration**: Rate-Limiting, Retry-Logik, Fallback-Strategien
- **Datenbank-Design**: TimescaleDB Hypertables für effiziente Zeitreihen
- **Numerische Verfahren**: Power-Law Interpolation für Windprofile, MLE-Parameterschätzung
- **Statistische Analyse**: Verteilungsanalyse, Parameterschätzung, Goodness-of-Fit Tests
- **Interface-Based Design**: Generische Interfaces für erweiterbare Architektur
- **Strategy Pattern**: Austauschbare Algorithmen und Implementierungen
- **Software-Architektur**: Layered Architecture, Dependency Injection, Code-Refactoring
- **Separation of Concerns**: Modularisierung und Entkopplung von Komponenten

## 🛠️ Tech Stack

- **Go 1.25+**: Hauptprogrammiersprache
- **PostgreSQL + TimescaleDB**: Zeitreihen-Datenbank
- **Open-Meteo API**: Wetterdatenquelle (Archive, Historical Forecast, Forecast)
- **pgx/v5**: PostgreSQL Treiber
- **gonum/stat**: Statistische Funktionen (Korrelation, Quantile, etc.)
- **gonum/mathext**: Spezielle mathematische Funktionen (Gamma, Digamma, etc.)
- **go-echarts**: Interaktive Chart-Bibliothek
- **gopkg.in/yaml.v3**: YAML-Konfiguration
- **Interface-Based Design**: Generische Verteilungsarchitektur

## 🛣️ Roadmap

### ✅ Phase 1: Dateninfrastruktur (Abgeschlossen)
- Open-Meteo API Integration mit Rate-Limiting
- Concurrency Pipeline für parallele Datenabfrage
- PostgreSQL/TimescaleDB Schema
- Power-Law Interpolation

### ✅ Phase 2: Statistische Analyse (Abgeschlossen)
- Weibull-Parameterschätzung (MLE mit Location-Parameter)
- Goodness-of-Fit Tests (KS-Test, AIC, BIC, RMSE)
- Deskriptive Statistik & Korrelation (Pearson, Spearman, Kendall)
- Power-Law Modellvalidierung

### ✅ Phase 3: Visualisierung (Abgeschlossen)
- Weibull-PDF/CDF Plots
- Histogramme & Vergleichs-Charts
- Heatmaps für Standortvergleiche
- Interaktive HTML-Dashboards

### ✅ Phase 4: Architektur-Refactoring (Abgeschlossen)
- Interface-basierte Verteilungsarchitektur (`ContinuousDistribution`, `Fitter`)
- Generische Datenbereinigung und Statistik-Funktionen
- Automatische Modellauswahl (`SelectBestModel`)
- Generische Visualisierungskomponenten
- Unterstützung mehrerer Verteilungen (Weibull, Gamma, Log-Normal)

### 📋 Phase 5: Erweiterte Verteilungen (Geplant)
- Rayleigh-Verteilung (Sonderfall von Weibull)
- Normalverteilung (als Referenz)
- Log-Logistic-Verteilung
- Generalized Extreme Value (GEV) für Extremwertanalyse

### 📋 Phase 6: Windenergie-Berechnungen (Geplant)
- Windenergie-Potenzialberechnung
- Capacity Factor Analyse
- Power Curve Integration

## ⚠️ Limitierungen & Disclaimer

- **API-Abhängigkeit**: Abhängig von Open-Meteo API Verfügbarkeit und Limits
- **Datenqualität**: Abhängig von Wetterdaten und Modellqualität
- **Modell-Simplifikationen**: Realer Wind ist komplexer als statistische Verteilungen
- **Automatische Modellauswahl**: Basiert auf AIC, aber keine Garantie für das "wahre" Modell
- **Kein professionelles Tool**: Nicht für Investitionsentscheidungen geeignet

## 📄 Lizenz

Dieses Projekt dient Lernzwecken und kann frei verwendet und modifiziert werden.