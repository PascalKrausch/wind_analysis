package visualization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"sort"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

const masterDashboardNoDataMessage = "Für %s kann keine Windgeschwindigkeit oder Hellmann Exponent geladen oder dargestellt werden"

type masterDashboardTimelinePayload struct {
	Wind     map[string][]masterDashboardTimelinePoint `json:"wind"`
	Hellmann map[string][]masterDashboardTimelinePoint `json:"hellmann"`
}

type masterDashboardTimelinePoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

type masterDashboardDistributionChart struct {
	chart    components.Charter
	location string
}

// PlotMasterDashboard rendert Zeitreihen, Validierungsmetriken und Verteilungs-Fits
// für die im Datenmodell enthaltenen Standorte in eine HTML-Datei.
func PlotMasterDashboard(master *MasterDashboardData, outputPath string) error {
	if master == nil {
		return fmt.Errorf("Master-Dashboard-Daten dürfen nicht nil sein")
	}
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("Ausgabepfad für Master-Dashboard darf nicht leer sein")
	}

	payload, err := buildMasterDashboardTimelinePayload(master)
	if err != nil {
		return err
	}

	page := components.NewPage().SetPageTitle("Windanalyse – Master-Dashboard").SetLayout(components.PageNoneLayout)
	windChart := buildMasterDashboardTimelineChart(master, "Windgeschwindigkeit über Zeit", "Windgeschwindigkeit (m/s)")
	hellmannChart := buildMasterDashboardTimelineChart(master, "Hellmann-Exponent über Zeit", "Hellmann-Exponent α")
	page.AddCharts(windChart, hellmannChart)

	distributionCharts := make([]masterDashboardDistributionChart, 0, len(master.GetAllDistributionInputs())*2)
	for _, input := range master.GetAllDistributionInputs() {
		distributionCharts = append(distributionCharts,
			masterDashboardDistributionChart{chart: BuildDistributionHistogramPDFChart(input), location: input.LocationName},
			masterDashboardDistributionChart{chart: BuildDistributionCDFChart(input), location: input.LocationName},
		)
	}
	for _, item := range distributionCharts {
		page.AddCharts(item.chart)
	}

	var rendered bytes.Buffer
	if err := page.Render(&rendered); err != nil {
		return fmt.Errorf("Master-Dashboard konnte nicht gerendert werden: %w", err)
	}

	content := rendered.String()
	content, err = addMasterDashboardControls(content, master, payload, windChart.ChartID, hellmannChart.ChartID)
	if err != nil {
		return err
	}
	content = addMasterDashboardSections(content, hellmannChart.ChartID, distributionCharts)
	content = addDashboardChartMetadataByID(content, windChart.ChartID, "", "wind")
	content = addDashboardChartMetadataByID(content, hellmannChart.ChartID, "", "hellmann")
	for _, item := range distributionCharts {
		content = addDashboardChartMetadata(content, item.chart, item.location, "distribution")
	}
	content = addMasterDashboardValidationTable(content, master)

	if err := RenderHTMLToFile(content, outputPath); err != nil {
		return fmt.Errorf("Master-Dashboard konnte nicht gespeichert werden: %w", err)
	}
	return nil
}

func buildMasterDashboardTimelinePayload(master *MasterDashboardData) (masterDashboardTimelinePayload, error) {
	payload := masterDashboardTimelinePayload{
		Wind:     make(map[string][]masterDashboardTimelinePoint, len(master.Locations)),
		Hellmann: make(map[string][]masterDashboardTimelinePoint, len(master.Locations)),
	}
	for _, location := range master.Locations {
		for _, point := range master.WindSpeedData[location] {
			if !point.Time.IsZero() && isFiniteNumber(point.Value) {
				payload.Wind[location] = append(payload.Wind[location], masterDashboardTimelinePoint{
					Time:  point.Time.UTC().Format("2006-01-02T15:04:05.000000000Z"),
					Value: point.Value,
				})
			}
		}
		for _, point := range master.HellmannData[location] {
			if !point.Time.IsZero() && isFiniteNumber(point.Value) {
				payload.Hellmann[location] = append(payload.Hellmann[location], masterDashboardTimelinePoint{
					Time:  point.Time.UTC().Format("2006-01-02T15:04:05.000000000Z"),
					Value: point.Value,
				})
			}
		}
	}
	for _, location := range master.Locations {
		sort.Slice(payload.Wind[location], func(i, j int) bool { return payload.Wind[location][i].Time < payload.Wind[location][j].Time })
		sort.Slice(payload.Hellmann[location], func(i, j int) bool { return payload.Hellmann[location][i].Time < payload.Hellmann[location][j].Time })
	}
	return payload, nil
}

func buildMasterDashboardTimelineChart(
	master *MasterDashboardData,
	title, yAxis string,
) *charts.Line {
	config := DefaultChartConfig()
	config.Title = title
	config.XAxisName = "Zeit"
	config.YAxisName = yAxis
	config.EnableDataZoom = true
	config.Height = "500px"
	line := CreateLineChart(config)
	line.SetXAxis([]string{})
	for _, location := range master.Locations {
		line.AddSeries(location, []opts.LineData{})
	}
	return line
}

func addMasterDashboardControls(content string, master *MasterDashboardData, payload masterDashboardTimelinePayload, windChartID, hellmannChartID string) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("Master-Dashboard-Zeitreihen konnten nicht serialisiert werden: %w", err)
	}

	var controls strings.Builder
	controls.WriteString(`<main class="dashboard">`)
	controls.WriteString(`<h1>Windanalyse – Master-Dashboard</h1>`)
	controls.WriteString(`<section class="filters"><label for="location-filter">Standorte (Mehrfachauswahl möglich)</label>`)
	controls.WriteString(`<details id="location-filter"><summary>Alle Standorte (` + fmt.Sprint(len(master.Locations)) + `)</summary><div class="location-options">`)
	for _, location := range master.Locations {
		controls.WriteString(`<label><input type="checkbox" checked value="`)
		controls.WriteString(html.EscapeString(location))
		controls.WriteString(`">`)
		controls.WriteString(html.EscapeString(location))
		controls.WriteString(`</label>`)
	}
	controls.WriteString(`</div></details><p class="hint">Über die Kontrollkästchen können einzelne oder mehrere Standorte verglichen werden.</p>`)
	controls.WriteString(`<label for="date-start">Zeitraum für Zeitreihen</label><div class="date-filter"><input id="date-start" type="date"><span>bis</span><input id="date-end" type="date"></div></section>`)
	controls.WriteString(`<section><h2>Validierungsmetriken</h2><div id="validation-table"></div></section>`)
	controls.WriteString(`<section id="timeseries-section"><h2>Zeitreihen</h2><div id="wind-error" class="notice" hidden></div><div id="hellmann-error" class="notice" hidden></div>`)
	controls.WriteString(`<style>
body{margin:0;background:#f5f7fa;color:#1f2937;font-family:Arial,sans-serif}
.dashboard{max-width:1400px;margin:0 auto;padding:24px}
.dashboard section{background:#fff;border-radius:8px;margin:18px 0;padding:18px;box-shadow:0 1px 4px #0001}
.filters{display:grid;grid-template-columns:1fr;gap:8px}
.filters details{position:relative;max-width:520px}
.filters summary{cursor:pointer;border:1px solid #b8c0cc;border-radius:4px;padding:10px;background:#fff}
.location-options{display:grid;gap:6px;max-height:260px;overflow:auto;padding:10px;border:1px solid #d9dee7;background:#fff}
.location-options label{display:flex;align-items:center;gap:8px}
.location-options input{margin:0}
.hint{margin:0;color:#667085;font-size:.9em}
.date-filter{display:flex;align-items:center;gap:10px}
.notice{margin:10px 0;padding:12px;background:#fff4e5;border-left:4px solid #e69b00}
table{border-collapse:collapse;width:100%}th,td{border:1px solid #d9dee7;padding:8px;text-align:left}
th{background:#eef2f7}tr:nth-child(even){background:#f8fafc}
.container{margin:10px auto;width:100%}
.item{max-width:100%}
[hidden]{display:none!important}
@media(max-width:600px){.dashboard{padding:10px}.date-filter{flex-wrap:wrap}}
</style>`)
	controls.WriteString(`<script>window.masterDashboardData=`)
	controls.Write(data)
	controls.WriteString(`;</script>`)

	if !strings.Contains(content, "<body>") {
		return "", fmt.Errorf("Master-Dashboard-Seite enthält kein body-Element")
	}
	content = strings.Replace(content, "<body>", "<body>\n"+controls.String(), 1)
	return strings.Replace(content, "</body>", masterDashboardFilterJS(windChartID, hellmannChartID)+"</body>", 1), nil
}

func addDashboardChartMetadata(content string, chart components.Charter, location, kind string) string {
	return addDashboardChartMetadataByID(content, chartID(chart), location, kind)
}

func chartID(chart components.Charter) string {
	switch value := chart.(type) {
	case *charts.Line:
		return value.ChartID
	case *charts.Bar:
		return value.ChartID
	default:
		return ""
	}
}

func addDashboardChartMetadataByID(content, id, location, kind string) string {
	if id == "" {
		return content
	}
	old := `<div class="item" id="` + id + `"`
	new := `<div class="item" data-dashboard-kind="` + html.EscapeString(kind) + `" data-dashboard-location="` + html.EscapeString(location) + `" id="` + id + `"`
	return strings.Replace(content, old, new, 1)
}

func addMasterDashboardValidationTable(content string, master *MasterDashboardData) string {
	var table strings.Builder
	rows := master.GetAllValidationRows()
	if len(rows) == 0 {
		table.WriteString(`<p class="hint">Für die verfügbaren Standorte liegen keine Validierungsmetriken vor.</p>`)
	} else {
		table.WriteString(`<table><thead><tr><th>Standort</th><th>Höhe [m]</th><th>MAE</th><th>RMSE</th><th>Korrelation</th><th>Stichproben</th></tr></thead><tbody>`)
		for _, row := range rows {
			table.WriteString(`<tr data-validation-location="`)
			table.WriteString(html.EscapeString(row.Location))
			table.WriteString(`"><td>`)
			table.WriteString(html.EscapeString(row.Location))
			table.WriteString(`</td><td>`)
			table.WriteString(fmt.Sprintf("%.0f", row.HeightM))
			table.WriteString(`</td><td>`)
			table.WriteString(fmt.Sprintf("%.4f", row.MAE))
			table.WriteString(`</td><td>`)
			table.WriteString(fmt.Sprintf("%.4f", row.RMSE))
			table.WriteString(`</td><td>`)
			table.WriteString(fmt.Sprintf("%.4f", row.Correlation))
			table.WriteString(`</td><td>`)
			table.WriteString(fmt.Sprint(row.SampleCount))
			table.WriteString(`</td></tr>`)
		}
		table.WriteString(`</tbody></table>`)
	}
	return strings.Replace(content, `<div id="validation-table"></div>`, `<div id="validation-table">`+table.String()+`</div>`, 1)
}

func addMasterDashboardSections(content, hellmannChartID string, distributions []masterDashboardDistributionChart) string {
	distributionSection := `</section><section id="distribution-section"><h2>Verteilungs-Fits</h2><div id="distribution-empty" class="notice" hidden>Für die ausgewählten Standorte sind keine Verteilungs-Fits verfügbar.</div>`
	content = insertAfterDashboardChart(content, hellmannChartID, distributionSection)

	if len(distributions) == 0 {
		return strings.Replace(content, `</body>`, `</section></main></body>`, 1)
	}
	lastChartID := chartID(distributions[len(distributions)-1].chart)
	return insertAfterDashboardChart(content, lastChartID, `</section></main>`)
}

func insertAfterDashboardChart(content, id, markup string) string {
	position := strings.Index(content, `<div class="item" id="`+id+`"`)
	if position < 0 {
		return content
	}
	end := strings.Index(content[position:], "</script>")
	if end < 0 {
		return content
	}
	end += position + len("</script>")
	return content[:end] + markup + content[end:]
}

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func masterDashboardFilterJS(windChartID, hellmannChartID string) string {
	return fmt.Sprintf(`<script>
(function(){
 const data=window.masterDashboardData;
 const selector=document.getElementById("location-filter");
 const checkboxes=Array.from(selector.querySelectorAll('input[type="checkbox"]'));
 const start=document.getElementById("date-start");
 const end=document.getElementById("date-end");
 const locations=checkboxes.map(o=>o.value);
 const ids={wind:%q,hellmann:%q};
 const noDataMessage=%q;
 const chartFor=id=>echarts.getInstanceByDom(document.getElementById(id));
 const selected=()=>checkboxes.filter(o=>o.checked).map(o=>o.value);
 function update(){
  const chosen=selected(), from=start.value, to=end.value;
  selector.querySelector("summary").textContent=chosen.length===locations.length?"Alle Standorte ("+locations.length+")":chosen.length?chosen.length+" Standorte ausgewählt":"Keine Standorte ausgewählt";
  for(const metric of ["wind","hellmann"]){
   const chart=chartFor(ids[metric]);
   const pointsByLocation={}; locations.forEach(name=>pointsByLocation[name]=new Map((data[metric][name]||[]).map(point=>[point.time,point.value])));
   const dates=Array.from(new Set(chosen.flatMap(name=>(data[metric][name]||[]).map(point=>point.time)))).sort().filter(value=>(!from||value.slice(0,10)>=from)&&(!to||value.slice(0,10)<=to));
   const valid=chosen.filter(name=>dates.some(value=>pointsByLocation[name].has(value)));
   const error=document.getElementById(metric+"-error");
   const item=document.getElementById(ids[metric]).parentElement;
   if(chart){
    chart.setOption({xAxis:[{data:dates.map(value=>new Date(value).toLocaleString("de-DE"))}],series:locations.map(name=>({name,data:dates.map(value=>pointsByLocation[name].get(value)??null)}))},{notMerge:false});
    const selectedLegend={}; chosen.forEach(name=>selectedLegend[name]=true); locations.filter(name=>!chosen.includes(name)).forEach(name=>selectedLegend[name]=false);
    chart.setOption({legend:{selected:selectedLegend}});
   }
   item.style.display=valid.length?"":"none";
   const missing=chosen.filter(name=>!valid.includes(name));
   error.hidden=missing.length===0&&chosen.length>0;
   error.textContent=chosen.length?missing.map(name=>noDataMessage.replace("%%s",name)).join(" "):"Bitte mindestens einen Standort auswählen.";
  }
  document.querySelectorAll("[data-validation-location]").forEach(row=>row.hidden=!chosen.includes(row.dataset.validationLocation));
  const fitCharts=Array.from(document.querySelectorAll('[data-dashboard-kind="distribution"]'));
  fitCharts.forEach(item=>item.parentElement.style.display=chosen.includes(item.dataset.dashboardLocation)?"":"none");
  document.getElementById("distribution-empty").hidden=chosen.some(name=>fitCharts.some(item=>item.dataset.dashboardLocation===name));
 }
 selector.addEventListener("change",update);start.addEventListener("change",update);end.addEventListener("change",update);
 update();
})();
</script>`, windChartID, hellmannChartID, masterDashboardNoDataMessage)
}
