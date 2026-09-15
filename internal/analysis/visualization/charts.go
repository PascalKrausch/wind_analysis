package visualization

import (
	"bytes"
	"html"
	"io"
	"os"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// ChartConfig enthält Konfigurationsoptionen für Charts
type ChartConfig struct {
	Title          string
	Subtitle       string
	XAxisName      string
	YAxisName      string
	Width          string
	Height         string
	Theme          string // "default", "light", "dark", "westeros", etc.
	Responsive     bool   // Responsive Design aktivieren
	EnableZoom     bool   // Zoom-Funktion aktivieren
	EnableDataZoom bool   // DataZoom für Zeitachsen aktivieren
}

func DefaultChartConfig() ChartConfig {
	return ChartConfig{
		Title:          "Chart",
		Subtitle:       "",
		XAxisName:      "X",
		YAxisName:      "Y",
		Width:          "1200px",
		Height:         "800px",
		Theme:          "westeros",
		Responsive:     true,
		EnableZoom:     true,
		EnableDataZoom: false,
	}
}

// CreateLineChart erstellt ein Line-Chart mit den angegebenen Konfigurationen
func CreateLineChart(config ChartConfig) *charts.Line {
	line := charts.NewLine()
	initOpts := opts.Initialization{Theme: config.Theme}

	if config.Responsive {
		initOpts.Width = "100%"
		// Wichtig: nicht 100%, sondern echte Höhe verwenden
		if config.Height != "" {
			initOpts.Height = config.Height
		} else {
			initOpts.Height = "800px"
		}
	} else {
		initOpts.Width = config.Width
		initOpts.Height = config.Height
	}

	// Globale Optionen setzen
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithInitializationOpts(initOpts),
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
	)

	// Toolbox mit Zoom-Funktion hinzufügen
	if config.EnableZoom {
		line.SetGlobalOptions(charts.WithToolboxOpts(opts.Toolbox{
			Show: opts.Bool(true),
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  opts.Bool(true),
					Title: "Speichern",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show:  opts.Bool(true),
					Title: map[string]string{"zoom": "Zoom", "back": "Zurück"},
				},
				Restore: &opts.ToolBoxFeatureRestore{
					Show:  opts.Bool(true),
					Title: "Reset",
				},
			},
		}))
	}

	// DataZoom für Zeitachsen hinzufügen
	if config.EnableDataZoom {
		line.SetGlobalOptions(charts.WithDataZoomOpts(opts.DataZoom{
			Type:  "slider",
			Start: 0,
			End:   100,
		}))
	}

	return line
}

// CreateScatterChart erstellt ein Scatter-Chart mit den angegebenen Konfigurationen
func CreateScatterChart(config ChartConfig) *charts.Scatter {
	scatter := charts.NewScatter()
	initOpts := opts.Initialization{Theme: config.Theme}

	if config.Responsive {
		initOpts.Width = "100%"
		if config.Height != "" {
			initOpts.Height = config.Height
		} else {
			initOpts.Height = "800px"
		}
	} else {
		initOpts.Width = config.Width
		initOpts.Height = config.Height
	}

	// Globale Optionen setzen
	scatter.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithInitializationOpts(initOpts),
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "item",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
	)

	// Toolbox mit Zoom-Funktion hinzufügen
	if config.EnableZoom {
		scatter.SetGlobalOptions(charts.WithToolboxOpts(opts.Toolbox{
			Show: opts.Bool(true),
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  opts.Bool(true),
					Title: "Speichern",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show:  opts.Bool(true),
					Title: map[string]string{"zoom": "Zoom", "back": "Zurück"},
				},
				Restore: &opts.ToolBoxFeatureRestore{
					Show:  opts.Bool(true),
					Title: "Reset",
				},
			},
		}))
	}

	return scatter
}

// CreateBarChart erstellt ein Bar-Chart mit den angegebenen Konfigurationen
func CreateBarChart(config ChartConfig) *charts.Bar {
	bar := charts.NewBar()
	initOpts := opts.Initialization{Theme: config.Theme}

	if config.Responsive {
		initOpts.Width = "100%"
		if config.Height != "" {
			initOpts.Height = config.Height
		} else {
			initOpts.Height = "800px"
		}
	} else {
		initOpts.Width = config.Width
		initOpts.Height = config.Height
	}

	// Globale Optionen setzen
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithInitializationOpts(initOpts),
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
	)

	// Toolbox mit Zoom-Funktion hinzufügen
	if config.EnableZoom {
		bar.SetGlobalOptions(charts.WithToolboxOpts(opts.Toolbox{
			Show: opts.Bool(true),
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  opts.Bool(true),
					Title: "Speichern",
				},
				DataZoom: &opts.ToolBoxFeatureDataZoom{
					Show:  opts.Bool(true),
					Title: map[string]string{"zoom": "Zoom", "back": "Zurück"},
				},
				Restore: &opts.ToolBoxFeatureRestore{
					Show:  opts.Bool(true),
					Title: "Reset",
				},
			},
		}))
	}

	return bar
}

// CreateHeatMap erstellt ein HeatMap-Chart mit den angegebenen Konfigurationen
func CreateHeatMap(config ChartConfig) *charts.HeatMap {
	heatMap := charts.NewHeatMap()

	heatMap.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  config.Width,
			Height: config.Height,
			Theme:  config.Theme,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "item",
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: opts.Bool(true),
			Min:        0,
			Max:        1,
			InRange: &opts.VisualMapInRange{
				Color: []string{"#50a3ba", "#eac736", "#d94e5d"},
			},
		}),
	)

	return heatMap
}

// CreateBoxPlot erstellt ein BoxPlot-Chart mit den angegebenen Konfigurationen
func CreateBoxPlot(config ChartConfig) *charts.BoxPlot {
	boxPlot := charts.NewBoxPlot()

	boxPlot.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    config.Title,
			Subtitle: config.Subtitle,
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  config.Width,
			Height: config.Height,
			Theme:  config.Theme,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: config.XAxisName,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: config.YAxisName,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "item",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
		}),
	)

	return boxPlot
}

// ChartRenderer ist ein Interface für Charts, die gerendert werden können
type ChartRenderer interface {
	Render(w io.Writer) error
}

// RenderToFile rendert ein Chart in eine HTML-Datei
func RenderToFile(chart ChartRenderer, filename string) error {
	var buf bytes.Buffer
	if err := chart.Render(&buf); err != nil {
		return err
	}

	htmlContent := buf.String()

	// Nur CSS injizieren (kein JS, das Höhe auf 100% erzwingt)
	styleTag := `
<style>
html, body {
	margin: 0;
	padding: 0;
	width: 100%;
}
body > div:first-child {
	width: 100% !important;
	min-height: 700px !important;
}
</style>
`

	if strings.Contains(htmlContent, "</head>") {
		htmlContent = strings.Replace(htmlContent, "</head>", styleTag+"</head>", 1)
	} else {
		htmlContent = styleTag + htmlContent
	}

	return RenderHTMLToFile(htmlContent, filename)
}

// RenderHTMLToFile rendert beliebiges HTML in eine Datei (z. B. Tabellen-Übersichten).
func RenderHTMLToFile(htmlContent, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(htmlContent)
	return err
}

// RenderToWriter rendert ein Chart in einen io.Writer
func RenderToWriter(chart ChartRenderer, writer io.Writer) error {
	return chart.Render(writer)
}

// AddLineSeries fügt eine Linien-Serie zu einem Line-Chart hinzu
func AddLineSeries(line *charts.Line, name string, xData []string, yData []float64) {
	line.SetXAxis(xData).
		AddSeries(name, convertToLineData(yData))
}

// AddScatterSeries fügt eine Scatter-Serie zu einem Scatter-Chart hinzu
func AddScatterSeries(scatter *charts.Scatter, name string, data []opts.ScatterData) {
	scatter.AddSeries(name, data)
}

// AddBarSeries fügt eine Bar-Serie zu einem Bar-Chart hinzu
func AddBarSeries(bar *charts.Bar, name string, xData []string, yData []float64) {
	bar.SetXAxis(xData).
		AddSeries(name, convertToBarData(yData))
}

// Hilfsfunktionen zur Datenkonvertierung
func convertToLineData(data []float64) []opts.LineData {
	items := make([]opts.LineData, 0, len(data))
	for _, d := range data {
		items = append(items, opts.LineData{Value: d})
	}
	return items
}

func convertToBarData(data []float64) []opts.BarData {
	items := make([]opts.BarData, 0, len(data))
	for _, d := range data {
		items = append(items, opts.BarData{Value: d})
	}
	return items
}

// ConvertToScatterData konvertiert zwei Arrays in Scatter-Daten
func ConvertToScatterData(xData, yData []float64) []opts.ScatterData {
	if len(xData) != len(yData) {
		return nil
	}

	items := make([]opts.ScatterData, 0, len(xData))
	for i := range xData {
		items = append(items, opts.ScatterData{
			Value: []interface{}{xData[i], yData[i]},
		})
	}
	return items
}

// SetCommonSeriesOptions setzt häufige Serien-Optionen für Line-Charts
func SetCommonSeriesOptions(line *charts.Line) {
	line.SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
	)
}

func RenderTable(outputPath, title, subtitle string, headers []string, rows [][]string) error {
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"de\"><head><meta charset=\"utf-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title>")
	b.WriteString(`<style>
body{font-family:Arial,sans-serif;margin:24px}
h1{margin:0 0 8px} p{margin:0 0 16px;color:#555}
table{border-collapse:collapse;width:100%}
th,td{border:1px solid #ddd;padding:8px;text-align:left}
th{background:#f4f4f4}
tr:nth-child(even){background:#fafafa}
</style></head><body>`)
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</h1>")
	if subtitle != "" {
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(subtitle))
		b.WriteString("</p>")
	}
	b.WriteString("<table><thead><tr>")
	for _, h := range headers {
		b.WriteString("<th>")
		b.WriteString(html.EscapeString(h))
		b.WriteString("</th>")
	}
	b.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		b.WriteString("<tr>")
		for _, cell := range row {
			b.WriteString("<td>")
			b.WriteString(html.EscapeString(cell))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")

	return RenderHTMLToFile(b.String(), outputPath)
}
