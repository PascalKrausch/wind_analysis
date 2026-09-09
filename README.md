# Wind Analysis - Wind Data Pipeline & Statistical Analysis

**⚠️ Hinweis:** Lernprojekt zur praktischen Anwendung von statistischen Methoden und numerischen Verfahren in Go. Nicht für produktive Windenergie-Planung geeignet.

## 🎯 Projektziel

Statistische Analyse von Windgeschwindigkeitsdaten mittels Weibull-Verteilung, Parameterschätzung und Performance-optimierte Datenverarbeitung.

## 🏗️ Architektur

```
wind-analysis/
├── cmd/
│   ├── fetcher/         # Einfacher Fetcher für einzelne Standorte
│   └── analyser/        # Statistische Analysen (in Entwicklung)
├── internal/
│   ├── client/          # Open-Meteo API Client mit Rate-Limiting
│   ├── database/        # PostgreSQL/TimescaleDB Integration
│   └── analysis/
│       └── interpolation/  # Power-Law Windprofil-Interpolation
├── pipeline/            # Concurrency-Engine für parallele Datenabfrage
├── models/              # Datenstrukturen & Konfiguration
├── config.yaml          # Standorte & Pipeline-Konfiguration
└── docker-compose.yml   # Datenbank-Setup
```

## 🚀 Funktionalität

### Implementiert

- **Open-Meteo API Integration**: Abruf von Winddaten für verschiedene Höhen (10m, 80m, 100m, 120m, 180m, 200m)
- **Automatische API-Auswahl**: Intelligente Wahl zwischen Forecast, Historical Forecast und Archive API
- **Concurrency Pipeline**: Parallele Datenabfrage mit konfigurierbaren Worker-Routinen
- **Rate-Limiting**: Konfigurierbare API-Rate-Limits für stabile Datenabfrage
- **Power-Law Interpolation**: Windprofil-Interpolation zwischen verschiedenen Höhen
- **Batch-Verarbeitung**: Effizientes Speichern großer Datensätze via PostgreSQL COPY
- **Datenbank-Speicherung**: PostgreSQL mit TimescaleDB für effiziente Zeitreihen-Abfragen

### In Planung

- **Weibull-Parameterschätzung**: Shape (k) und Scale (λ) Parameter via MLE und Method of Moments
- **Goodness-of-Fit Tests**: KS-Test, Chi-Square für Modellvalidierung
- **Windenergie-Berechnung**: Theoretische Energieerträge und Capacity Factors
- **Statistische Vergleiche**: Wasserstein-Distanz für zeitliche Veränderungen

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


### Pipeline (Parallele Abfrage mehrerer Standorte)

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

## 🎓 Lernziele

- **Concurrency Patterns**: Fan-Out/Fan-In Pipeline mit Worker-Pools
- **API-Integration**: Rate-Limiting, Retry-Logik, Fallback-Strategien
- **Datenbank-Design**: TimescaleDB Hypertables für effiziente Zeitreihen
- **Numerische Verfahren**: Power-Law Interpolation für Windprofile
- **Statistische Analyse**: Weibull-Verteilung, Parameterschätzung, Goodness-of-Fit Tests

## 🛠️ Tech Stack

- **Go 1.20+**: Hauptprogrammiersprache
- **PostgreSQL + TimescaleDB**: Zeitreihen-Datenbank
- **Open-Meteo API**: Wetterdatenquelle (Archive, Historical Forecast, Forecast)
- **pgx/v5**: PostgreSQL Treiber
- **Geplant**: gonum/stat, gonum/dist, gonum/optimize

## 🛣️ Roadmap

### ✅ Phase 1: Dateninfrastruktur (Abgeschlossen)
- Open-Meteo API Integration mit Rate-Limiting
- Concurrency Pipeline für parallele Datenabfrage
- PostgreSQL/TimescaleDB Schema
- Power-Law Interpolation

### 🔄 Phase 2: Statistische Analyse (In Entwicklung)
- Weibull-Parameterschätzung (MLE, Method of Moments)
- Goodness-of-Fit Tests (KS-Test, Chi-Square)
- Wasserstein-Distanz für zeitliche Vergleiche

### 📋 Phase 3: Windenergie-Berechnungen (Geplant)
- Windenergie-Potenzialberechnung
- Capacity Factor Analyse
- Power Curve Integration

### 📋 Phase 4: Visualisierung (Geplant)
- Weibull-PDF Plots, Q-Q Plots
- Windrosen
- Zeitreihen-Visualisierungen

## ⚠️ Limitierungen & Disclaimer

- **API-Abhängigkeit**: Abhängig von Open-Meteo API Verfügbarkeit und Limits
- **Datenqualität**: Abhängig von Wetterdaten und Modellqualität
- **Modell-Simplifikationen**: Realer Wind ist komplexer als reine Weibull-Verteilung
- **Kein professionelles Tool**: Nicht für Investitionsentscheidungen geeignet

## 📄 Lizenz

Dieses Projekt dient Lernzwecken und kann frei verwendet und modifiziert werden.