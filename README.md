# Wind Analysis - Weibull Distribution & Wind Energy Potential

**⚠️ Hinweis:** Dies ist ein Lernprojekt zur praktischen Anwendung von statistischen Methoden und numerischen Verfahren in Go. Es dient primär der technischen Explorerung und nicht der produktiven Verwendung für echte Windenergie-Planung.

## 🎯 Projektziel

Dieses Projekt konzentriert sich auf die statistische Analyse von Windgeschwindigkeitsdaten mittels Weibull-Verteilung. Es demonstriert die Implementierung von Parameterschätzungsverfahren, statistischen Tests und deren Visualisierung in Go.

## 🌪️ Fokus-Themen

- **Weibull-Verteilung**: Formparameter (k) und Skalenparameter (λ) Schätzung
- **Parameterschätzung**: Maximum Likelihood Estimation (MLE) und Method of Moments
- **Goodness-of-Fit Tests**: Kolmogorov-Smirnov, Chi-Square Tests
- **Windenergie-Potenzial**: Theoretische Energieerträge basierend auf Windverteilung
- **Statistische Vergleiche**: Vergleich von Windverteilungen zwischen Standorten und Zeitperioden
- **Wasserstein-Distanz**: Zeitliche Veränderung von Windgeschwindigkeitsverteilungen

## 🛠️ Technischer Fokus

Das Projekt konzentriert sich auf folgende Go-Konzepte:

- **Numerische Verfahren**: Optimierungsalgorithmen, numerische Integration
- **Statistische Bibliotheken**: gonum/stat, gonum/dist, gonum/optimize
- **Statistische Tests**: Hypothesis Testing, p-Values, Confidence Intervals
- **Erweiterte Visualisierung**: PDF/CDF Plots, Q-Q Plots, Windrosen
- **Performance-Optimierung**: Effiziente Verarbeitung großer Datensätze
- **API-Integration**: Open-Meteo API für Wetterdaten
- **Datenbank-Speicherung**: PostgreSQL mit TimescaleDB für Zeitreihen

## 🏗️ Architektur

```
wind-analysis/
├── cmd/
│   ├── fetcher/         # Winddaten von Open-Meteo API holen und speichern
│   └── analyzer/        # Hauptanwendung für Windanalyse (in Planung)
├── internal/
│   ├── client/          # Open-Meteo API Client
│   ├── database/        # PostgreSQL/TimescaleDB Integration
│   ├── distribution/    # Weibull-Verteilung Implementierung (geplant)
│   ├── estimation/      # Parameterschätzungsverfahren (geplant)
│   ├── testing/        # Statistical Tests & Goodness-of-Fit (geplant)
│   ├── energy/         # Windenergie-Berechnungen (geplant)
│   └── comparison/     # Vergleichende Analysen (geplant)
├── models/
│   └── models.go       # Wind-Datenstrukturen
└── .env                 # Datenbank-Konfiguration
```

## 🚀 Funktionalität

### Aktuell Implementiert

- **Open-Meteo API Integration**: Abruf von Winddaten für verschiedene Höhen (10m, 80m, 100m, 120m, 180m, 200m)
- **Automatische API-Auswahl**: Intelligente Wahl zwischen Forecast, Historical Forecast und Archive API
- **Datenbank-Speicherung**: PostgreSQL mit TimescaleDB für effiziente Zeitreihen-Speicherung
- **Höhenprofil-Unterstützung**: Umgang mit verschiedenen Höhenprofilen je nach API und Zeitraum
- **Batch-Verarbeitung**: Effizientes Speichern großer Datensätze

### Geplant

- **Weibull-Parameterschätzung**: Berechnung von Shape (k) und Scale (λ) Parametern
- **Multiple Schätzmethoden**: Method of Moments und Maximum Likelihood
- **Goodness-of-Fit Tests**: Bewertung der Anpassungsgüte
- **Windenergie-Berechnung**: Theoretische Energieerträge und Capacity Factors
- **Standort-Vergleiche**: Statistische Tests zwischen verschiedenen Städten
- **Wasserstein-Distanz**: Zeitliche Veränderung von Windverteilungen
- **Erweiterte Visualisierung**: Weibull-PDF, Q-Q Plots, Windrosen

### Datenquelle

Dieses Projekt verwendet die **Open-Meteo API** als primäre Datenquelle:

- **Archive API**: Historische Daten von 1940 bis heute (10m, 100m Höhen)
- **Historical Forecast API**: Daten ab ca. 2022 mit vollem Höhenprofil
- **Forecast API**: Aktuelle Daten mit `past_days` Parameter für sehr aktuelle Daten

## 📋 Voraussetzungen

- Go 1.20+
- PostgreSQL mit TimescaleDB Extension
- Docker & Docker Compose (für lokale Datenbank)
- Umgebungsvariablen in `.env` Datei

## 🔧 Installation

1. Repository klonen
2. Abhängigkeiten installieren:
   ```bash
   go mod download
   ```
3. Datenbank starten:
   ```bash
   docker-compose up -d
   ```
4. `.env` Datei erstellen:
   ```bash
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=your_password
   DB_NAME=wind_analysis
   ```

## 🚀 Nutzung

### Winddaten abrufen und speichern

```bash
# Standardverwendung (Berlin, 2 Tage)
go run cmd/fetcher/main.go

# Mit benutzerdefinierten Parametern
go run cmd/fetcher/main.go \
  -location "Hamburg" \
  -lat 53.5511 \
  -lon 9.9937 \
  -start "2024-01-01" \
  -end "2024-01-07"

# Alle verfügbaren Optionen
go run cmd/fetcher/main.go -help
```

### API-Endpoint-Logik

Der Fetcher wählt automatisch den passenden API-Endpoint:

- **Sehr aktuell (letzte 3 Monate)**: Forecast API mit `past_days` für volles Höhenprofil
- **2022-2024**: Historical Forecast API für volles Höhenprofil  
- **Vor 2022**: Archive API mit 10m/100m + Power Law Interpolation (geplant)

### Datenbank-Struktur

- **locations**: Geografische Standorte mit Koordinaten
- **wind_logs**: Zeitreihen-Daten als TimescaleDB Hypertable für effiziente Abfragen

## 🎓 Lernziele

Durch dieses Projekt werden folgende Konzepte praktisch angewendet:

### Aktuell:
- **API-Integration**: Nutzung von Open-Meteo API für Wetterdaten
- **Datenbank-Design**: PostgreSQL mit TimescaleDB für Zeitreihen
- **Batch-Verarbeitung**: Effiziente Speicherung großer Datensätze
- **API-Endpoint-Logik**: Intelligente Wahl zwischen verschiedenen API-Endpunkten

### Geplant:
- **Wahrscheinlichkeitsverteilungen**: Weibull, Log-Normal, Gamma
- **Parameterschätzung**: MLE, Method of Moments, Bayesian Methods
- **Statistische Tests**: KS-Test, Chi-Square, Anderson-Darling
- **Wasserstein-Distanz**: Optimal Transport für Verteilungsvergleiche
- **Numerische Optimierung**: Gradient Descent, Newton-Raphson
- **Spezielle Funktionen**: Gamma-Funktion, Incomplete Gamma
- **Statistische Bibliotheken**: Effiziente Nutzung von gonum
- **Performance**: Memory-Management für große Datensätze
- **Visualisierung**: PDF/CDF, Q-Q, P-P Diagramme, Windrosen

## 📊 Beispielausgabe

### Aktuell (Fetcher)
```
🌪️  Hole Winddaten für Berlin (52.5200, 13.4050) von 2024-01-01 bis 2024-01-02...
✅ Verfügbare Höhen: [10m 100m]
📊 Anzahl der Datenpunkte: 48
✅ Erfolgreich 48 Winddatensätze für Berlin in der Datenbank gespeichert!
```

### Geplant (Analyzer)
- **Weibull-Parameter**: Shape (k) und Scale (λ) mit Konfidenzintervallen
- **Goodness-of-Fit**: Statistiken und p-Values für verschiedene Tests
- **Windenergie-Potenzial**: Theoretische Jahreserträge in kWh
- **Vergleichsanalysen**: Statistische Signifikanz von Standort-Unterschieden
- **Wasserstein-Distanz**: Zeitliche Veränderung von Windverteilungen
- **Visualisierungen**: Weibull-PDF mit empirischen Daten, Q-Q Plots

## 🔬 Wissenschaftlicher Kontext

Die Weibull-Verteilung ist der Standard für die Modellierung von Windgeschwindigkeiten in der Windenergie-Industrie. Dieses Projekt demonstriert:

- **Praktische Anwendung**: Wie theoretische Statistik auf reale Daten angewendet wird
- **Modellvalidierung**: Wie man überprüft, ob ein Modell wirklich passt
- **Entscheidungsunterstützung**: Wie statistische Ergebnisse zu praktischen Empfehlungen führen
- **Datenintegration**: Umgang mit verschiedenen Wetterdatenquellen und API-Endpunkten

## ⚠️ Limitierungen

### Aktuell:
- **API-Abhängigkeit**: Abhängig von Open-Meteo API Verfügbarkeit und Limits
- **Höhenprofil-Einschränkungen**: Archive API bietet nur 10m und 100m Daten
- **Datenverzögerung**: Archive API hat bis zu 5 Tage Verzögerung
- **Keine Interpolation**: Windprofil-Interpolation noch nicht implementiert

### Allgemein:
- **Datenqualität**: Abhängig von der Qualität der Wetterdaten und Modelle
- **Modell-Simplifikationen**: Realer Wind ist komplexer als reine Weibull-Verteilung
- **Geografische Faktoren**: Topografie, Orografie werden nicht berücksichtigt
- **Zeitliche Variationen**: Saisonale Effekte werden vereinfacht behandelt

## 🔮 Potenziale Erweiterungen

Als Lernprojekt bietet sich das Projekt für folgende Erweiterungen an:

### Kurzfristig:
- **Power Law Interpolation**: Windprofil-Interpolation zwischen verschiedenen Höhen
- **Verbesserte API-Auswahl**: Automatische Wechsellogik zwischen Forecast und Archive API
- **Datenvalidierung**: Qualitätstests für abgerufene Daten

### Mittelfristig:
- **Weibull-Analyse**: Vollständige Implementierung der statistischen Analysen
- **Wasserstein-Distanz**: Zeitliche Veränderung von Windverteilungen
- **Erweiterte Verteilungen**: Mixture Models, bivariate Weibull

### Langfristig:
- **Zeitreihen-Analyse**: ARIMA-Modelle für Windgeschwindigkeit
- **Machine Learning**: Neural Networks für Windprognose
- **GIS-Integration**: Geografische Visualisierung der Windpotenziale
- **Real-time Analysis**: Streaming-Datenverarbeitung für Windmessungen

## 🛠️ Tech Stack

### Backend
- **Go 1.20+**: Hauptprogrammiersprache
- **PostgreSQL**: Datenbank
- **TimescaleDB**: Erweiterung für Zeitreihen-Daten
- **pgx/v5**: PostgreSQL Treiber für Go

### API & Daten
- **Open-Meteo API**: Wetterdatenquelle
  - Archive API (historische Daten)
  - Historical Forecast API (ab 2022)
  - Forecast API (aktuelle Daten)

### Geplante Bibliotheken
- **gonum/stat**: Statistische Funktionen
- **gonum/dist**: Wahrscheinlichkeitsverteilungen
- **gonum/optimize**: Numerische Optimierung
- **gonum/interp**: Interpolationsmethoden

## 🛣️ Roadmap

### Phase 1: Dateninfrastruktur (✅ Abgeschlossen)
- [x] Open-Meteo API Integration
- [x] Datenbank-Schema mit TimescaleDB
- [x] Fetcher für Winddaten
- [x] Batch-Verarbeitung

### Phase 2: Windprofil-Interpolation (🔄 In Planung)
- [ ] Power Law Interpolation implementieren
- [ ] Logarithmische Interpolation als Alternative
- [ ] Automatische API-Endpoint-Auswahl
- [ ] Validierung der Interpolationsmethoden

### Phase 3: Statistische Analyse (📋 Geplant)
- [ ] Weibull-Parameterschätzung (MLE, Method of Moments)
- [ ] Goodness-of-Fit Tests (KS-Test, Chi-Square)
- [ ] Wasserstein-Distanz für zeitliche Vergleiche
- [ ] Konfidenzintervalle und Bootstrap-Methoden

### Phase 4: Windenergie-Berechnungen (📋 Geplant)
- [ ] Windenergie-Potenzialberechnung
- [ ] Capacity Factor Analyse
- [ ] Power Curve Integration
- [ ] Luftdichtekorrektur

### Phase 5: Visualisierung (📋 Geplant)
- [ ] Weibull-PDF Plots
- [ ] Q-Q Plots
- [ ] Windrosen
- [ ] Zeitreihen-Visualisierungen

## 📄 Lizenz

Dieses Projekt dient Lernzwecken und kann frei verwendet und modifiziert werden.

## ⚠️ Disclaimer

Dies ist **kein** professionelles Windenergie-Planungstool. Die Ergebnisse sollten nicht für tatsächliche Investitionsentscheidungen in der Windenergie verwendet werden. Das Projekt ist als technische Demonstration statistischer Methoden konzipiert.