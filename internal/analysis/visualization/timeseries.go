package visualization

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/opts"
)

// TimeSeriesPoint repräsentiert einen einzelnen Datenpunkt über die Zeit.
type TimeSeriesPoint struct {
	Time  time.Time
	Value float64
}

// LocationTimeSeries bündelt eine Zeitreihe für einen bestimmten Standort.
type LocationTimeSeries struct {
	LocationName string
	Points       []TimeSeriesPoint
}

type aggregatedLevelPayload struct {
	XAxis  []string         `json:"xAxis"`
	Series map[string][]any `json:"series"`
}

// PlotMultiLocationTimeline zeichnet den zeitlichen Verlauf beliebiger Kennzahlen für mehrere Standorte.
func PlotMultiLocationTimeline(seriesList []LocationTimeSeries, title, yAxisName, outputPath string) error {
	if len(seriesList) == 0 {
		return fmt.Errorf("keine Zeitreihendaten für Standortvergleich vorhanden")
	}

	levels := []string{AggregationHourly, AggregationDaily, AggregationWeekly, AggregationMonthly}
	payloadByLevel := make(map[string]aggregatedLevelPayload, len(levels))
	locationNames := make([]string, 0, len(seriesList))

	var minT, maxT time.Time
	for i, s := range seriesList {
		locationNames = append(locationNames, s.LocationName)
		for _, p := range s.Points {
			if i == 0 && p.Time.Before(minT) || (i == 0 && minT.IsZero()) {
				minT = p.Time
			}
			if maxT.IsZero() || p.Time.After(maxT) {
				maxT = p.Time
			}
		}
	}
	sort.Strings(locationNames)

	for _, level := range levels {
		allTimesMap := make(map[time.Time]struct{})
		seriesAgg := make(map[string][]TimeSeriesPoint, len(seriesList))

		for _, s := range seriesList {
			agg := AggregateTimeSeries(s.Points, level)
			seriesAgg[s.LocationName] = agg
			for _, p := range agg {
				allTimesMap[p.Time] = struct{}{}
			}
		}

		times := make([]time.Time, 0, len(allTimesMap))
		for t := range allTimesMap {
			times = append(times, t)
		}
		sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })

		xData := ConvertTimesToXAxisByLevel(times, level)
		payload := aggregatedLevelPayload{
			XAxis:  xData,
			Series: make(map[string][]any, len(seriesList)),
		}

		for _, s := range seriesList {
			valMap := make(map[time.Time]float64, len(seriesAgg[s.LocationName]))
			for _, p := range seriesAgg[s.LocationName] {
				valMap[p.Time] = p.Value
			}

			ySeries := make([]any, len(times))
			for i, t := range times {
				if v, ok := valMap[t]; ok {
					ySeries[i] = v
				} else {
					ySeries[i] = nil
				}
			}
			payload.Series[s.LocationName] = ySeries
		}
		payloadByLevel[level] = payload
	}

	totalDays := maxT.Sub(minT).Hours() / 24
	initialLevel := AggregationHourly
	switch {
	case totalDays > 730:
		initialLevel = AggregationMonthly
	case totalDays > 120:
		initialLevel = AggregationWeekly
	case totalDays > 21:
		initialLevel = AggregationDaily
	}

	cfg := DefaultChartConfig()
	cfg.Title = title
	cfg.XAxisName = "Zeit"
	cfg.YAxisName = yAxisName
	cfg.EnableDataZoom = true

	line := CreateLineChart(cfg)
	line.SetXAxis(payloadByLevel[initialLevel].XAxis)

	for _, name := range locationNames {
		raw := payloadByLevel[initialLevel].Series[name]
		ySeries := make([]opts.LineData, len(raw))
		for i, v := range raw {
			ySeries[i] = opts.LineData{Value: v}
		}
		line.AddSeries(name, ySeries)
	}

	jsPayload, err := json.Marshal(payloadByLevel)
	if err != nil {
		return fmt.Errorf("konnte Aggregationsdaten nicht serialisieren: %w", err)
	}

	SetCommonSeriesOptions(line)
	line.AddJSFuncs(getDynamicZoomJS(string(jsPayload), initialLevel))
	return RenderToFile(line, outputPath)
}

// getDynamicZoomJS liefert den JS-Event-Listener für Zoom + dynamische Aggregation.
func getDynamicZoomJS(aggJSON, initialLevel string) string {
	// Falls aggJSON leer ist, Fallback auf leeres Objekt
	if strings.TrimSpace(aggJSON) == "" {
		aggJSON = "{}"
	}

	jsTemplate := `
	(function() {
		var AGG = %s;
		var currentLevel = "%s";
		var timer = null;

		function axisFormatter(level) {
			if (level === "monthly") {
				return function(value) {
					return value || "";
				};
			}
			if (level === "weekly" || level === "daily") {
				return function(value) {
					if (typeof value !== "string") return value;
					return value.length >= 10 ? value.slice(5, 10) : value;
				};
			}
			return function(value) {
				if (typeof value !== "string") return value;
				if (value.length >= 16) {
					return value.slice(11, 16) + " " + value.slice(5, 10);
				}
				return value;
			};
		}

		function parseTs(v) {
			if (typeof v === "number") return v;
			if (typeof v !== "string") return NaN;
			var s = v;
			if (s.length === 7) s += "-01 00:00";
			if (s.length === 10) s += " 00:00";
			return Date.parse(s.replace(" ", "T"));
		}

		function pickLevel(rangeDays) {
			if (rangeDays > 730) return "monthly";
			if (rangeDays > 120) return "weekly";
			if (rangeDays > 21) return "daily";
			return "hourly";
		}

		function applyLevel(chart, level) {
			if (!AGG[level]) return;
			var opt = chart.getOption();
			var series = opt.series || [];
			for (var i = 0; i < series.length; i++) {
				var n = series[i].name;
				if (AGG[level].series && AGG[level].series[n]) {
					series[i].data = AGG[level].series[n];
				}
			}
			var dz = (opt.dataZoom && opt.dataZoom.length > 0) ? opt.dataZoom[0] : {start:0, end:100};

			chart.setOption({
				xAxis: [{
					data: AGG[level].xAxis,
					axisLabel: { formatter: axisFormatter(level) }
				}],
				series: series,
				dataZoom: [{ start: dz.start || 0, end: dz.end || 100 }]
			});

			currentLevel = level;
		}

		setTimeout(function() {
			var chartDom = document.querySelector("div[_echarts_instance_]");
			if (!chartDom) return;
			var chart = echarts.getInstanceByDom(chartDom);
			if (!chart) return;

			applyLevel(chart, currentLevel);

			chart.on("datazoom", function () {
				if (timer) clearTimeout(timer);
				timer = setTimeout(function () {
					var opt = chart.getOption();
					var x = (opt.xAxis && opt.xAxis[0] && opt.xAxis[0].data) ? opt.xAxis[0].data : [];
					if (x.length < 2) return;

					var dz = (opt.dataZoom && opt.dataZoom.length > 0) ? opt.dataZoom[0] : {start:0, end:100};
					var si = Math.max(0, Math.floor((dz.start || 0) / 100 * (x.length - 1)));
					var ei = Math.min(x.length - 1, Math.ceil((dz.end || 100) / 100 * (x.length - 1)));

					var t0 = parseTs(x[si]);
					var t1 = parseTs(x[ei]);
					if (isNaN(t0) || isNaN(t1) || t1 <= t0) return;

					var rangeDays = (t1 - t0) / (1000 * 60 * 60 * 24);
					var next = pickLevel(rangeDays);
					if (next !== currentLevel) applyLevel(chart, next);
				}, 120);
			});
		}, 300);
	})();`

	return fmt.Sprintf(jsTemplate, aggJSON, initialLevel)
}
