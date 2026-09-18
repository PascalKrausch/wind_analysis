# Wind Analysis - Wind Data Pipeline & Statistical Analysis

**⚠️ Hinweis:** Lernprojekt zur praktischen Anwendung von statistischen Methoden und numerischen Verfahren in Go. Nicht für produktive Windenergie-Planung geeignet.

## 🎯 Projektziel

Statistische Analyse von Windgeschwindigkeitsdaten mittels Weibull-Verteilung, Parameterschätzung und Performance-optimierte Datenverarbeitung.

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
│       ├── distribution/    # Weibull-Verteilung & Parameterschätzung
│       ├── validation/      # Validierungs-Orchestrierung
│       ├── weibull/         # Weibull-Verarbeitungslogik
│       └── visualization/   # Chart-Erstellung (go-echarts)
├── pipeline/            # Concurrency-Engine für parallele Datenabfrage
├── models/              # Datenstrukturen & Konfiguration
├── config.yaml          # Standorte & Pipeline-Konfiguration
└── docker-compose.yml   # Datenbank-Setup
```

## 🏛️ Architektur-Prinzipien

Das Projekt folgt etablierten Software-Architektur-Patterns:

- **Layered Architecture**: Klare Trennung zwischen Entry Points, Business Logic und Data Access
- **Domain-Driven Design**: Packages nach fachlichen Domänen strukturiert
- **Dependency Injection**: Konstruktoren nehmen Abhängigkeiten als Parameter für Testbarkeit
- **Context-First Pattern**: Alle asynchronen Funktionen verwenden `context.Context`
- **Pipeline Pattern**: Fan-Out/Fan-In für parallele Datenverarbeitung
- **Repository Pattern**: Datenbankzugriffe gekapselt in dedizierten Packages

## 🚀 Funktionalität

### Implementiert

- **Open-Meteo API Integration**: Abruf von Winddaten für verschiedene Höhen (10m, 80m, 100m, 120m, 180m, 200m)
- **Automatische API-Auswahl**: Intelligente Wahl zwischen Forecast, Historical Forecast und Archive API
- **Concurrency Pipeline**: Parallele Datenabfrage mit konfigurierbaren Worker-Routinen
- **Rate-Limiting**: Konfigurierbare API-Rate-Limits für stabile Datenabfrage
- **Power-Law Interpolation**: Windprofil-Interpolation zwischen verschiedenen Höhen (Hellmann-Exponenten)
- **Batch-Verarbeitung**: Effizientes Speichern großer Datensätze via PostgreSQL COPY
- **Datenbank-Speicherung**: PostgreSQL mit TimescaleDB für effiziente Zeitreihen-Abfragen
- **Weibull-Parameterschätzung**: Shape (k), Scale (λ) und Location (μ) Parameterschätzung via MLE
- **Goodness-of-Fit Tests**: KS-Test, AIC, BIC, RMSE für Modellvalidierung
- **Statistische Analyse**: Deskriptive Statistik, Korrelation (Pearson, Spearman, Kendall), Histogramme
- **Visualisierung**: Interaktive HTML-Charts (Histogramme, PDF/CDF, Heatmaps, Vergleiche)
- **Validierung**: Power-Law Modellvalidierung mit Fehleranalyse

### In Planung

- **Windenergie-Berechnung**: Theoretische Energieerträge und Capacity Factors
- **Statistische Vergleiche**: Wasserstein-Distanz für zeitliche Veränderungen
- **Erweiterte Visualisierungen**: Windrosen, Zeitreihen-Dashboards

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
- **Numerische Verfahren**: Power-Law Interpolation für Windprofile, Weibull-Parameterschätzung
- **Statistische Analyse**: Weibull-Verteilung, Parameterschätzung, Goodness-of-Fit Tests
- **Visualisierung**: Interaktive Charts mit go-echarts
- **Software-Architektur**: Layered Architecture, Dependency Injection, Code-Refactoring

## 🛠️ Tech Stack

- **Go 1.25+**: Hauptprogrammiersprache
- **PostgreSQL + TimescaleDB**: Zeitreihen-Datenbank
- **Open-Meteo API**: Wetterdatenquelle (Archive, Historical Forecast, Forecast)
- **pgx/v5**: PostgreSQL Treiber
- **gonum/stat**: Statistische Funktionen (Korrelation, Quantile, etc.)
- **go-echarts**: Interaktive Chart-Bibliothek
- **gopkg.in/yaml.v3**: YAML-Konfiguration

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

### 📋 Phase 4: Windenergie-Berechnungen (Geplant)
- Windenergie-Potenzialberechnung
- Capacity Factor Analyse
- Power Curve Integration

## ⚠️ Limitierungen & Disclaimer

- **API-Abhängigkeit**: Abhängig von Open-Meteo API Verfügbarkeit und Limits
- **Datenqualität**: Abhängig von Wetterdaten und Modellqualität
- **Modell-Simplifikationen**: Realer Wind ist komplexer als reine Weibull-Verteilung
- **Kein professionelles Tool**: Nicht für Investitionsentscheidungen geeignet

## 📄 Lizenz

Dieses Projekt dient Lernzwecken und kann frei verwendet und modifiziert werden.