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
	Wind          map[string][]masterDashboardTimelinePoint `json:"wind"`
	Hellmann      map[string][]masterDashboardTimelinePoint `json:"hellmann"`
	Distributions map[string][]masterDashboardDistributionData `json:"distributions"`
}

type masterDashboardDistributionData struct {
	Location string `json:"location"`
	HeightM  int    `json:"heightM"`
	Series   string `json:"series"`
}

type masterDashboardTimelinePoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

type masterDashboardDistributionChart struct {
	chart    components.Charter
	location string
	height   int
	series   string
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

	// Verteilungs-Charts lazy loading: Nur erstellen, aber nicht sofort zur Page hinzufügen
	// Sie werden erst später bei Bedarf gerendert
	distributionInputs := master.GetAllDistributionInputs()
	distributionCharts := make([]masterDashboardDistributionChart, 0, len(distributionInputs)*2)
	for _, input := range distributionInputs {
		distributionCharts = append(distributionCharts,
			masterDashboardDistributionChart{chart: BuildDistributionHistogramPDFChart(input), location: input.LocationName, height: input.HeightM, series: input.SeriesName},
			masterDashboardDistributionChart{chart: BuildDistributionCDFChart(input), location: input.LocationName, height: input.HeightM, series: input.SeriesName},
		)
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
	// Lazy Loading: Verteilungs-Charts werden später über JavaScript geladen
	content = addMasterDashboardValidationTable(content, master)

	if err := RenderHTMLToFile(content, outputPath); err != nil {
		return fmt.Errorf("Master-Dashboard konnte nicht gespeichert werden: %w", err)
	}
	return nil
}

func buildMasterDashboardTimelinePayload(master *MasterDashboardData) (masterDashboardTimelinePayload, error) {
	payload := masterDashboardTimelinePayload{
		Wind:          make(map[string][]masterDashboardTimelinePoint, len(master.Locations)),
		Hellmann:      make(map[string][]masterDashboardTimelinePoint, len(master.Locations)),
		Distributions: make(map[string][]masterDashboardDistributionData, len(master.Locations)),
	}

	totalDays := master.EndTime.Sub(master.StartTime).Hours() / 24
	locationCount := len(master.Locations)

	// Aggressiveres Downsampling: Basis-Sampling basierend auf Zeitraum
	sampleInterval := 1
	if totalDays > 365 {
		sampleInterval = 48 // Jeden 48. Punkt bei > 1 Jahr (war 24)
	} else if totalDays > 90 {
		sampleInterval = 24 // Jeden 24. Punkt bei > 3 Monaten (war 12)
	} else if totalDays > 30 {
		sampleInterval = 12 // Jeden 12. Punkt bei > 1 Monat (war 6)
	}

	// Zusätzliches Scaling basierend auf Standort-Anzahl
	// Bei vielen Standorten müssen wir noch aggressiver sein
	if locationCount > 10 {
		sampleInterval = sampleInterval * 2
	} else if locationCount > 5 {
		sampleInterval = sampleInterval * 3 / 2 // = sampleInterval * 1.5
	}

	// Sampling-Intervall runden und minimum 1
	sampleIntervalInt := int(sampleInterval)
	if sampleIntervalInt < 1 {
		sampleIntervalInt = 1
	}

	for _, location := range master.Locations {
		for i, point := range master.WindSpeedData[location] {
			if !point.Time.IsZero() && isFiniteNumber(point.Value) && i%sampleIntervalInt == 0 {
				payload.Wind[location] = append(payload.Wind[location], masterDashboardTimelinePoint{
					Time:  point.Time.UTC().Format("2006-01-02T15:04:05.000000000Z"),
					Value: point.Value,
				})
			}
		}
		for i, point := range master.HellmannData[location] {
			if !point.Time.IsZero() && isFiniteNumber(point.Value) && i%sampleIntervalInt == 0 {
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

	// Sammle Metadaten für Verteilungs-Charts (Lazy Loading)
	for _, location := range master.Locations {
		if inputs, ok := master.DistributionData[location]; ok {
			for _, input := range inputs {
				payload.Distributions[location] = append(payload.Distributions[location], masterDashboardDistributionData{
					Location: input.LocationName,
					HeightM:  input.HeightM,
					Series:   input.SeriesName,
				})
			}
		}
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
	distributionSection := `</section><section id="distribution-section"><h2>Verteilungs-Fits</h2><div id="distribution-empty" class="notice" hidden>Für die ausgewählten Standorte sind keine Verteilungs-Fits verfügbar.</div><div id="distribution-container"></div>`
	content = insertAfterDashboardChart(content, hellmannChartID, distributionSection)

	// Lazy Loading: Chart-Daten in verstecktem Container speichern
	if len(distributions) > 0 {
		var chartData strings.Builder
		chartData.WriteString(`<script id="distribution-charts-data" type="application/json">`)
		chartData.WriteString(`[`)
		for i, dist := range distributions {
			if i > 0 {
				chartData.WriteString(`,`)
			}
			chartData.WriteString(`{`)
			chartData.WriteString(`"location":"` + html.EscapeString(dist.location) + `",`)
			chartData.WriteString(`"height":` + fmt.Sprintf("%d", dist.height) + `,`)
			chartData.WriteString(`"series":"` + html.EscapeString(dist.series) + `",`)
			chartData.WriteString(`"chartType":"` + getChartType(dist.chart) + `",`)
			chartData.WriteString(`"chartID":"` + chartID(dist.chart) + `"`)
			chartData.WriteString(`}`)
		}
		chartData.WriteString(`]`)
		chartData.WriteString(`</script>`)
		content = strings.Replace(content, `</body>`, chartData.String()+`</body>`, 1)
	}

	return strings.Replace(content, `</body>`, `</section></main></body>`, 1)
}

func getChartType(chart components.Charter) string {
	switch chart.(type) {
	case *charts.Bar:
		return "histogram"
	default:
		return "cdf"
	}
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
 const currentLevel={wind:"hourly",hellmann:"hourly"};
 let timer=null;

 function aggregateData(points,level){
  if(!points||points.length===0)return[];
  const buckets=new Map();
  points.forEach(p=>{
   const t=new Date(p.time);
   let key;
   if(level==="monthly")key=new Date(t.getFullYear(),t.getMonth(),1).toISOString();
   else if(level==="weekly"){
    const d=new Date(t);
    const day=d.getDay();
    const diff=d.getDate()-day+(day===0?-6:1);
    key=new Date(d.setDate(diff)).toISOString().slice(0,10);
   }else if(level==="daily")key=t.toISOString().slice(0,10);
   else key=t.toISOString().slice(0,13);
   if(!buckets.has(key))buckets.set(key,{sum:0,count:0});
   const b=buckets.get(key);
   b.sum+=p.value;
   b.count++;
  });
  const result=[];
  buckets.forEach((v,k)=>{
   result.push({time:k,value:v.sum/v.count});
  });
  return result.sort((a,b)=>a.time.localeCompare(b.time));
 }

 function pickLevel(rangeDays){
  if(rangeDays>730)return"monthly";
  if(rangeDays>120)return"weekly";
  if(rangeDays>21)return"daily";
  return"hourly";
 }

 function updateLevel(chart,metric,dates,pointsByLocation,level){
  const allX=[],series={};
  locations.forEach(name=>{
   series[name]=[];
  });
  dates.forEach(d=>{
   allX.push(new Date(d).toLocaleString("de-DE"));
   locations.forEach(name=>{
    const points=pointsByLocation[name];
    const agg=aggregateData(points,level);
    const point=agg.find(p=>p.time===d);
    series[name].push(point?point.value:null);
   });
  });
  chart.setOption({xAxis:[{data:allX}],series:locations.map(name=>({name,data:series[name]}))},{notMerge:false});
  currentLevel[metric]=level;
 }

 function update(){
  const chosen=selected(), from=start.value, to=end.value;
  selector.querySelector("summary").textContent=chosen.length===locations.length?"Alle Standorte ("+locations.length+")":chosen.length?chosen.length+" Standorte ausgewählt":"Keine Standorte ausgewählt";
  for(const metric of ["wind","hellmann"]){
   const chart=chartFor(ids[metric]);
   const pointsByLocation={}; locations.forEach(name=>pointsByLocation[name]=data[metric][name]||[]);
   const dates=Array.from(new Set(chosen.flatMap(name=>(data[metric][name]||[]).map(point=>point.time)))).sort().filter(value=>(!from||value.slice(0,10)>=from)&&(!to||value.slice(0,10)<=to));
   const valid=chosen.filter(name=>dates.some(value=>pointsByLocation[name].some(p=>p.time===value)));
   const error=document.getElementById(metric+"-error");
   const item=document.getElementById(ids[metric]).parentElement;
   if(chart){
    const t0=new Date(dates[0]),t1=new Date(dates[dates.length-1]);
    const rangeDays=(t1-t0)/(86400000);
    const level=pickLevel(rangeDays);
    updateLevel(chart,metric,dates,pointsByLocation,level);
    const selectedLegend={}; chosen.forEach(name=>selectedLegend[name]=true); locations.filter(name=>!chosen.includes(name)).forEach(name=>selectedLegend[name]=false);
    chart.setOption({legend:{selected:selectedLegend}});
    chart.off("datazoom");
    chart.on("datazoom",function(){
     if(timer)clearTimeout(timer);
     timer=setTimeout(function(){
      const opt=chart.getOption();
      const x=opt.xAxis[0].data;
      if(x.length<2)return;
      const dz=opt.dataZoom[0];
      const si=Math.max(0,Math.floor((dz.start||0)/100*(x.length-1)));
      const ei=Math.min(x.length-1,Math.ceil((dz.end||100)/100*(x.length-1)));
      const t0=Date.parse(x[si]),t1=Date.parse(x[ei]);
      if(isNaN(t0)||isNaN(t1)||t1<=t0)return;
      const rangeDays=(t1-t0)/(86400000);
      const next=pickLevel(rangeDays);
      if(next!==currentLevel[metric])updateLevel(chart,metric,dates,pointsByLocation,next);
     },120);
    });
   }
   item.style.display=valid.length?"":"none";
   const missing=chosen.filter(name=>!valid.includes(name));
   error.hidden=missing.length===0&&chosen.length>0;
   error.textContent=chosen.length?missing.map(name=>noDataMessage.replace("%%s",name)).join(" "):"Bitte mindestens einen Standort auswählen.";
  }
  document.querySelectorAll("[data-validation-location]").forEach(row=>row.hidden=!chosen.includes(row.dataset.validationLocation));
  updateDistributionCharts(chosen);
 }
 selector.addEventListener("change",update);start.addEventListener("change",update);end.addEventListener("change",update);
 update();

 // Lazy Loading für Verteilungs-Charts
 function updateDistributionCharts(chosen){
  const container=document.getElementById("distribution-container");
  const emptyMsg=document.getElementById("distribution-empty");
  if(!container)return;

  // Container leeren
  container.innerHTML="";

  // Chart-Daten abrufen
  const chartDataScript=document.getElementById("distribution-charts-data");
  if(!chartDataScript){
   emptyMsg.hidden=false;
   return;
  }

  try{
   const chartsData=JSON.parse(chartDataScript.textContent);
   const filteredCharts=chartsData.filter(chart=>chosen.includes(chart.location));

   if(filteredCharts.length===0){
    emptyMsg.hidden=false;
    return;
   }

   emptyMsg.hidden=true;

   // Nur Charts für ausgewählte Standorte rendern
   filteredCharts.forEach(chartData=>{
    const wrapper=document.createElement("div");
    wrapper.className="item";
    wrapper.dataset.dashboardKind="distribution";
    wrapper.dataset.dashboardLocation=chartData.location;
    wrapper.style.margin="10px auto";
    wrapper.style.maxWidth="100%%";

    const title=document.createElement("h3");
    title.textContent=chartData.location+" ("+chartData.height+"m) - "+chartData.series+" "+(chartData.chartType==="histogram"?"Histogram":"CDF");
    title.style.marginTop="0";
    wrapper.appendChild(title);

    const chartDiv=document.createElement("div");
    chartDiv.id=chartData.chartID;
    chartDiv.style.width="100%%";
    chartDiv.style.height="400px";
    wrapper.appendChild(chartDiv);

    container.appendChild(wrapper);
   });

   // Charts initialisieren
   if(typeof window.distributionChartsInit==="function"){
    window.distributionChartsInit();
   }
  }catch(e){
   console.error("Fehler beim Laden der Verteilungs-Charts:",e);
   emptyMsg.hidden=false;
  }
 }

 // Intersection Observer für lazy loading beim Scrollen
 const observer=new IntersectionObserver((entries)=>{
  entries.forEach(entry=>{
   if(entry.isIntersecting){
    const section=document.getElementById("distribution-section");
    if(section&&!section.dataset.loaded){
     section.dataset.loaded="true";
     // Charts laden wenn Sektion sichtbar wird
     const chosen=selected();
     updateDistributionCharts(chosen);
    }
   }
 },{threshold:0.1});

 const distributionSection=document.getElementById("distribution-section");
 if(distributionSection){
  observer.observe(distributionSection);
 }
})();
</script>`, windChartID, hellmannChartID, masterDashboardNoDataMessage)
}
