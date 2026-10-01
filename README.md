# wind_analysis

Eine in **Go** entwickelte Datenpipeline zur Sammlung, Verarbeitung und statistischen Analyse historischer Winddaten.

Das Projekt kombiniert **konkurrenten API-Datenabruf, PostgreSQL/TimescaleDB, statistische Verteilungsmodelle und interaktive Visualisierung**.

Es handelt sich um ein eigenständiges Engineering-Projekt mit Fokus auf Datenpipelines, numerische Verfahren und eine wartbare Go-Architektur.

> **Hinweis:** Das Projekt dient der technischen Untersuchung, dem Lernen und der statistischen Analyse. Es ist **kein professionelles Werkzeug für Windenergieplanung oder Investitionsentscheidungen**.

---

## 🎯 Ziel des Projekts

Ziel ist der Aufbau einer reproduzierbaren Pipeline zur Analyse von Windgeschwindigkeiten an unterschiedlichen Standorten und Messhöhen.

Das System deckt den gesamten Ablauf ab:

```text
Open-Meteo API
      ↓
Konkurrenter Datenabruf
      ↓
Rate Limiting / Retries
      ↓
Datenbereinigung
      ↓
PostgreSQL / TimescaleDB
      ↓
Statistische Analyse
      ↓
Anpassung statistischer Modelle
      ↓
Modellbewertung
      ↓
Interaktive Visualisierung
```

Die Architektur ist bewusst so aufgebaut, dass sich Datenerfassung und statistische Analyse unabhängig voneinander weiterentwickeln lassen.

---

## 🚀 Was das Projekt zeigt

### Datenverarbeitung

* Konkurrierender Abruf von Wetterdaten
* Worker-Pool-basierte Verarbeitung
* Konfigurierbares API-Rate-Limiting
* Retry- und Fallback-Mechanismen
* Batch-Verarbeitung
* PostgreSQL-Bulk-Inserts
* Staging-Tabellen mit UPSERT-Verarbeitung
* Deduplizierung innerhalb von Batches
* Datenintegrität über Datenbank-Constraints

### Statistische Analyse

* Deskriptive Statistik
* Pearson-, Spearman- und Kendall-Korrelation
* Weibull-Verteilung
* Gamma-Verteilung
* Lognormalverteilung
* Maximum-Likelihood-Schätzung
* AIC / BIC
* Kolmogorov-Smirnov-Test
* RMSE-basierte Bewertung
* Automatische Auswahl des passenden Modells

### Numerische Verfahren

* Interpolation von Windprofilen über ein Power-Law-Modell
* Höhenabhängige Analyse der Windgeschwindigkeit
* Validierung von Interpolationsmodellen
* Bewusster Umgang mit meteorologischen Modelldaten und deren Gitterpunkten

### Go-Architektur

* Interface-basierte Architektur
* Dependency Injection
* Schichtenorientierte Paketstruktur
* Pipeline-orientierte Nebenläufigkeit
* Trennung von Datenzugriff und Analyse
* Schrittweise Überführung von fachlich spezialisierter Logik in generische Komponenten

---

## 🏗️ Architektur

```text
wind_analysis/
│
├── cmd/
│   ├── fetcher/
│   │   └── main.go
│   └── analyser/
│       └── main.go
│
├── internal/
│   ├── client/
│   │   └── Open-Meteo API-Client
│   │
│   ├── database/
│   │   └── PostgreSQL / TimescaleDB-Zugriff
│   │
│   ├── analysis/
│   │   ├── interpolation/
│   │   ├── statistics/
│   │   ├── distribution/
│   │   ├── validation/
│   │   ├── weibull/
│   │   └── visualization/
│   │
│   └── utils/
│
├── pipeline/
│   └── Engine für konkurrierende Datenerfassung
│
├── models/
│   └── Konfiguration und Domänenmodelle
│
├── config.yaml
├── docker-compose.yml
├── go.mod
└── README.md
```

Die Trennung soll die wichtigsten Verantwortlichkeiten unabhängig voneinander halten:

* API-Kommunikation
* Pipeline-Steuerung
* Persistenz
* statistische Analyse
* Visualisierung

---

## ⚙️ Pipeline und Nebenläufigkeit

Der Fetcher verwendet eine konfigurierbare Worker-Pipeline, um mehrere Standorte gleichzeitig zu verarbeiten.

```text
                 ┌── Worker 1 ──┐
Standorte ───────┼── Worker 2 ──┼──────► Datenbank
                 ├── Worker 3 ──┤
                 └── Worker 4 ──┘
```

Anzahl der Worker, Batch-Größe und API-Rate-Limit sind konfigurierbar.

Beispiel:

```yaml
pipeline:
  concurrency: 4
  batch_size: 1000
  rate_limit_rps: 4
```

Dadurch kann der Durchsatz angepasst werden, ohne den eigentlichen Anwendungscode ändern zu müssen.

---

## 🗄️ Datenbankarchitektur

Für die Zeitreihendaten wird **PostgreSQL mit TimescaleDB** verwendet.

Größere Datenmengen werden nicht einzeln pro Zeile geschrieben. Stattdessen nutzt die Pipeline einen Staging-Ansatz:

```text
Eingehender Batch
      ↓
PostgreSQL COPY
      ↓
Staging-Tabelle
      ↓
UPSERT
      ↓
Produktive Zeitreihentabelle
```

Damit werden die Vorteile von Bulk Loading mit kontrollierter Konfliktbehandlung und Datenkonsistenz kombiniert.

Zusätzlich verhindern Unique Constraints, dass doppelte Messungen in die Haupttabelle gelangen.

---

## 🧠 Generische Architektur für statistische Verteilungen

Ein zentrales Refactoring des Projekts war die Entwicklung von einer zunächst auf die Weibull-Verteilung zugeschnittenen Analyse hin zu einer generischen, interface-basierten Architektur.

### Zentrales Interface

```go
type ContinuousDistribution interface {
    PDF(x float64) float64
    CDF(x float64) float64
    LogPDF(x float64) float64
    Params() []float64
}
```

### Interface für das Fitting

```go
type Fitter interface {
    Fit(data []float64) (ContinuousDistribution, error)
    Name() string
}
```

Aktuell sind unter anderem folgende Modelle integriert:

* Weibull
* Gamma
* Lognormal

Eine weitere Verteilung kann ergänzt werden, indem die entsprechenden Interfaces implementiert und der Fitter in die Modellauswahl integriert wird.

Dadurch bleibt die eigentliche Analyse unabhängig von den Details eines einzelnen statistischen Modells.

---

## 📊 Automatische Modellauswahl

Der Analysator kann mehrere Kandidaten vergleichen und anhand von **AIC** das passendste Modell auswählen.

Beispiel:

```go
bestDist, metrics, err := distribution.SelectBestModel(data)
```

Zusätzlich werden unter anderem folgende Kennzahlen berechnet:

* AIC
* BIC
* KS-Statistik
* RMSE

Dadurch können mehrere Modelle über denselben Bewertungsprozess miteinander verglichen werden.

---

## 🌬️ Interpolation von Windprofilen

Für Datensätze mit unterschiedlichen verfügbaren Messhöhen wird ein Power-Law-Modell verwendet, um Windgeschwindigkeiten zwischen Höhen abzuschätzen.

Die Interpolationslogik ist von der statistischen Verteilungsanalyse getrennt, sodass beide Bereiche unabhängig voneinander weiterentwickelt und überprüft werden können.

Zusätzlich werden die tatsächlichen Gitterkoordinaten und die vom Wetterdatenanbieter gelieferte Gitterhöhe gespeichert.

Dadurch lässt sich unterscheiden zwischen:

```text
angeforderte Koordinate
        vs.
tatsächlich verwendeter meteorologischer Gitterpunkt
```

und diese Differenz wird nicht stillschweigend ignoriert.

---

## 🌍 Datenquelle

Die Pipeline nutzt die **Open-Meteo API** und wählt abhängig vom angefragten Zeitraum automatisch zwischen den verfügbaren Endpunkten.

Unterstützt werden:

* Forecast-Daten
* historische Forecast-Daten
* Archivdaten

Die Datenerfassung kann für mehrere konfigurierbare Standorte und Höhen erfolgen.

---

## 📈 Visualisierung

Der Analysator erzeugt interaktive HTML-Ausgaben, unter anderem:

* Histogramme von Windgeschwindigkeiten
* Wahrscheinlichkeitsdichtefunktionen
* kumulative Verteilungsfunktionen
* Modellvergleiche
* Standortvergleiche
* Heatmaps
* Analysen von Windprofilen
* Validierungsdiagramme

Die erzeugten Dateien werden im Verzeichnis `output` abgelegt.

---

## 🧪 Beispielanalyse

Nach der Datenerfassung kann die Analyse gestartet werden:

```bash
go run cmd/analyser/main.go
```

Der Analysator liest die konfigurierte Datenbasis ein und erzeugt die entsprechenden statistischen Auswertungen und Visualisierungen.

---

## ⚡ Performance und Skalierung

Die Architektur berücksichtigt, dass die externe API der wichtigste limitierende Faktor für den Datendurchsatz ist.

Daher liegt der Fokus auf:

* parallelen Requests
* explizitem API-Rate-Limiting
* Batch-Verarbeitung
* PostgreSQL `COPY`
* Staging-Tabellen mit UPSERT
* Deduplizierung auf Datenbankebene

Mit der aktuellen Konfiguration ist die Datenbank auf effiziente Verarbeitung größerer Batches ausgelegt, während der externe API-Abruf den wesentlichen Flaschenhals darstellt.

---

## 🛠️ Tech-Stack

### Sprache

* **Go 1.25+**

### Daten & Persistenz

* **PostgreSQL**
* **TimescaleDB**
* **pgx/v5**

### Statistik & Mathematik

* **gonum/stat**
* **gonum/mathext**

### Visualisierung

* **go-echarts**

### Konfiguration

* **YAML (`gopkg.in/yaml.v3`)**

### Externe API

* **Open-Meteo**

---

## 📦 Voraussetzungen

* Go 1.25+
* Docker
* Docker Compose
* PostgreSQL mit TimescaleDB

Die enthaltene Docker-Konfiguration ist der einfachste Weg, die Datenbank lokal zu starten.

---

## 🔧 Einrichtung

### 1. Repository klonen

```bash
git clone https://github.com/PascalKrausch/wind_analysis.git
cd wind_analysis
```

### 2. Abhängigkeiten herunterladen

```bash
go mod download
```

### 3. Datenbank starten

```bash
docker compose up -d
```

### 4. Analyse konfigurieren

Über `config.yaml` lassen sich unter anderem folgende Parameter festlegen:

* Standorte
* Zeitraum
* Anzahl der Worker
* Batch-Größe
* API-Rate-Limit

Beispiel:

```yaml
locationlist:
  - name: Husum
    latitude: 54.4878
    longitude: 9.0556

  - name: Harz
    latitude: 51.7967
    longitude: 10.6206

timeframe:
  start: "2022-01-01"
  end: "2026-09-19"
  chunk_years: 2

pipeline:
  concurrency: 4
  batch_size: 1000
  rate_limit_rps: 4
```

### 5. Daten abrufen

```bash
go run cmd/fetcher/main.go -config config.yaml
```

### 6. Analyse starten

```bash
go run cmd/analyser/main.go
```

Die erzeugten Ausgaben werden unter folgendem Pfad gespeichert:

```text
./output
```

---

## 🔍 Datenbank untersuchen

Gitter-Metadaten können direkt in PostgreSQL untersucht werden.

Beispiel:

```sql
SELECT
    location_name,
    AVG(grid_elevation) AS avg_grid_elevation,
    MIN(grid_elevation) AS min_grid_elevation,
    MAX(grid_elevation) AS max_grid_elevation,
    COUNT(*) AS sample_count
FROM wind_logs
GROUP BY location_name;
```

Unterschiede zwischen angeforderter und tatsächlich verwendeter Gitterposition können beispielsweise so analysiert werden:

```sql
SELECT
    location_name,
    AVG(ABS(grid_latitude - latitude)) AS avg_lat_diff,
    AVG(ABS(grid_longitude - longitude)) AS avg_lon_diff
FROM wind_logs
GROUP BY location_name;
```

---

## 🔄 Entwicklung der Architektur

Die statistische Analyse wurde ursprünglich stark auf die Weibull-Verteilung zugeschnitten.

Anschließend wurde die Architektur schrittweise in Richtung generischer Komponenten refaktoriert:

```text
Weibull-spezifische Analyse
          ↓
Gemeinsame Interfaces
          ↓
Generische Verteilungsschicht
          ↓
Mehrere Fitting-Implementierungen
          ↓
Automatische Modellauswahl
```

Die bestehende Weibull-Funktionalität bleibt dabei erhalten, während weitere Verteilungen über dieselbe Architektur integriert werden können.

Das Refactoring erfolgt bewusst schrittweise und nicht als vollständiger Rewrite.

---

## 📚 Technische Entscheidungen

Einige Entscheidungen sind für die Architektur besonders wichtig:

### Interfaces statt fest verdrahteter Verteilungslogik

Das Verhalten statistischer Verteilungen wird über Interfaces beschrieben. Dadurch hängt die Analysepipeline nicht von einem einzigen Modell ab.

### Kontrollierte Nebenläufigkeit statt unbegrenzter Parallelität

Mehrere Worker erhöhen den Durchsatz, werden aber mit konfigurierbarem Rate-Limiting kombiniert, um externe API-Grenzen einzuhalten.

### Bulk Writes statt Einzel-Inserts

Größere historische Datensätze werden in Batches verarbeitet, um den Overhead einzelner Datenbankoperationen zu reduzieren.

### Mehrere Validierungsmetriken statt einer einzigen Annahme

Verschiedene statistische Kennzahlen ermöglichen einen nachvollziehbareren Modellvergleich.

### Sichtbare Metadaten statt versteckter Approximationen

Tatsächliche Gitterkoordinaten und Höhen werden gespeichert, damit Annahmen und mögliche Abweichungen später überprüfbar bleiben.

---

## 🚧 Aktueller Stand

### Abgeschlossen

* Open-Meteo-Integration
* Rate-limited Concurrent Ingestion
* PostgreSQL / TimescaleDB
* Batch-Verarbeitung
* Staging-Tabellen mit UPSERT
* Deduplizierung und Unique Constraints
* Windprofil-Interpolation
* Statistische Analyse
* Weibull-Fitting
* Gamma-Fitting
* Lognormal-Fitting
* Goodness-of-Fit-Bewertung
* Automatische Modellauswahl
* Interaktive Visualisierung
* Generische Verteilungsarchitektur

### Geplant

* Weitere Wahrscheinlichkeitsverteilungen
* Extremwertanalyse
* Berechnungen zur Windenergienutzung
* Kapazitätsfaktor-Analyse
* Weitere statistische Vergleichsmethoden
* Erweiterte Windvisualisierungen
* Weitere Überführung älterer Weibull-spezifischer Logik in die generische Architektur

---

## ⚠️ Grenzen des Projekts

Das Projekt ist als Engineering- und Statistikprojekt gedacht und nicht als professionelles Planungssystem.

Wichtige Einschränkungen:

* Ergebnisse hängen von Qualität und Verfügbarkeit der zugrunde liegenden Wetterdaten ab
* meteorologische Modelldaten repräsentieren Gitterzellen und nicht exakte Messpunkte
* die Gitterhöhe kann von der tatsächlichen Geländehöhe abweichen
* Power-Law-Interpolation ist eine vereinfachte Beschreibung der realen atmosphärischen Bedingungen
* kein statistisches Modell garantiert eine perfekte Beschreibung der Realität
* API-Verfügbarkeit und Rate-Limits begrenzen die Datenerfassung

Die Ergebnisse sollten daher nicht unmittelbar für Investitionsentscheidungen oder professionelle Windenergieplanung verwendet werden.

---

## 📌 Fokus des Projekts

Der Schwerpunkt von `wind_analysis` liegt darauf, zu untersuchen, wie sich ein komplexeres Datenverarbeitungsproblem in Go strukturieren lässt und dabei:

* Nebenläufigkeit explizit bleibt
* Datenzugriff gekapselt ist
* statistische Algorithmen erweiterbar bleiben
* Performance messbar wird
* Annahmen sichtbar bleiben
* zukünftige Erweiterungen möglich sind

Das Projekt ist bewusst mehr als ein einfacher API-Client: Es verbindet Datenerfassung, Persistenz, numerische Verarbeitung, statistische Modellierung und Visualisierung in einer durchgängigen Pipeline.
