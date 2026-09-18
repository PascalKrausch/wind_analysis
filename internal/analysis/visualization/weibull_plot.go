package visualization

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"wind_analysis/internal/analysis/statistics"
	"wind_analysis/models"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type WeibullPlotInput struct {
	SeriesName   string
	LocationName string
	HeightM      int

	// Bereits berechnete Statistik-/Fit-Ergebnisse.
	Analysis models.WeibullAnalysisResult

	// Roh- oder Aufbereitungsdaten für Visualisierungen.
	Histogram    []statistics.Bin
	EmpiricalCDF []statistics.CDFPoint
	FittedPDF    []statistics.DensityPoint
	FittedCDF    []statistics.CDFPoint
}

func NewWeibullDashboard(inputs []WeibullPlotInput) *components.Page {
	page := components.NewPage()
	page.PageTitle = "Weibull Analysis"

	for _, in := range inputs {
		page.AddCharts(
			BuildWeibullHistogramPDFChart(in),
			BuildWeibullCDFChart(in),
		)
	}

	if len(inputs) > 1 {
		page.AddCharts(
			BuildWeibullComparisonChart(inputs),
			BuildWeibullMetricsChart(inputs),
		)
	}

	return page
}

func PlotWeibullDashboard(inputs []WeibullPlotInput, outputPath string) error {
	return savePage(NewWeibullDashboard(inputs), outputPath)
}

func PlotWeibullComparisonOverview(inputs []WeibullPlotInput, comparisons []models.WeibullComparisonResult, outputPath string) error {
	page := components.NewPage()
	page.PageTitle = "Weibull Comparison"

	if len(inputs) > 0 {
		page.AddCharts(
			BuildWeibullComparisonChart(inputs),
			BuildWeibullMetricsChart(inputs),
		)
	}

	if len(comparisons) > 0 {
		page.AddCharts(BuildWeibullComparisonHeatmap(comparisons))
	}

	return savePage(page, outputPath)
}

func savePage(page *components.Page, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return page.Render(f)
}

// SavePage speichert eine go-echarts Page in eine Datei (public für generische Nutzung).
func SavePage(page *components.Page, outputPath string) error {
	return savePage(page, outputPath)
}

func BuildWeibullHistogramPDFChart(in WeibullPlotInput) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    chartTitle("Weibull Histogramm + PDF", in),
			Subtitle: "Histogramm (Anzahl) und Modell-PDF (Dichte) im selben Koordinatensystem",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Windgeschwindigkeit [m/s]"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Anzahl"}),
	)
	bar.ExtendYAxis(opts.YAxis{Name: "Dichte"})

	x, centers := histogramXAxisAndCenters(in.Histogram)

	hist := make([]opts.BarData, 0, len(in.Histogram))
	for _, b := range in.Histogram {
		hist = append(hist, opts.BarData{Value: b.Count})
	}
	bar.SetXAxis(x).AddSeries("Histogramm", hist)

	pdf := densityAtXVals(in.FittedPDF, centers)
	line := charts.NewLine()
	line.SetXAxis(x).AddSeries(
		"Weibull-PDF",
		toLineData(pdf),
		charts.WithLineChartOpts(opts.LineChart{YAxisIndex: 1, Smooth: opts.Bool(true)}),
	)

	bar.Overlap(line)
	return bar
}

// BuildDistributionHistogramPDFChart erstellt ein Histogramm + PDF Chart für jede Verteilung.
func BuildDistributionHistogramPDFChart(in DistributionPlotInput) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    DistributionChartTitle(fmt.Sprintf("%s Histogramm + PDF", in.ModelName), in),
			Subtitle: "Histogramm (Anzahl) und Modell-PDF (Dichte) im selben Koordinatensystem",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Wert"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Anzahl"}),
	)
	bar.ExtendYAxis(opts.YAxis{Name: "Dichte"})

	x, centers := histogramXAxisAndCenters(in.Histogram)

	hist := make([]opts.BarData, 0, len(in.Histogram))
	for _, b := range in.Histogram {
		hist = append(hist, opts.BarData{Value: b.Count})
	}
	bar.SetXAxis(x).AddSeries("Histogramm", hist)

	pdf := densityAtXVals(in.FittedPDF, centers)
	line := charts.NewLine()
	line.SetXAxis(x).AddSeries(
		fmt.Sprintf("%s-PDF", in.ModelName),
		toLineData(pdf),
		charts.WithLineChartOpts(opts.LineChart{YAxisIndex: 1, Smooth: opts.Bool(true)}),
	)

	bar.Overlap(line)
	return bar
}

func BuildWeibullCDFChart(in WeibullPlotInput) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    chartTitle("Weibull CDF", in),
			Subtitle: "Empirische CDF vs. Modell-CDF",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Windgeschwindigkeit [m/s]"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Kumulative Wahrscheinlichkeit", Min: 0, Max: 1}),
	)

	empX, empY := cdfSeries(in.EmpiricalCDF)
	modX, modY := cdfSeries(in.FittedCDF)
	x := mergeSortedXAxis(empX, modX)

	line.SetXAxis(x).
		AddSeries("Empirisch", toLineData(alignSeries(x, empX, empY))).
		AddSeries("Weibull-Modell", toLineData(alignSeries(x, modX, modY)))

	return line
}

// BuildDistributionCDFChart erstellt ein CDF Chart für jede Verteilung.
func BuildDistributionCDFChart(in DistributionPlotInput) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    DistributionChartTitle(fmt.Sprintf("%s CDF", in.ModelName), in),
			Subtitle: "Empirische CDF vs. Modell-CDF",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Wert"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Kumulative Wahrscheinlichkeit", Min: 0, Max: 1}),
	)

	empX, empY := cdfSeries(in.EmpiricalCDF)
	modX, modY := cdfSeries(in.FittedCDF)
	x := mergeSortedXAxis(empX, modX)

	line.SetXAxis(x).
		AddSeries("Empirisch", toLineData(alignSeries(x, empX, empY))).
		AddSeries(fmt.Sprintf("%s-Modell", in.ModelName), toLineData(alignSeries(x, modX, modY)))

	return line
}

func BuildWeibullMetricsChart(inputs []WeibullPlotInput) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Weibull Fit-Güte",
			Subtitle: "AIC, BIC, KS und RMSE je Serie",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Serie"}),
	)

	x := make([]string, 0, len(inputs))
	aic := make([]opts.LineData, 0, len(inputs))
	bic := make([]opts.LineData, 0, len(inputs))
	ks := make([]opts.LineData, 0, len(inputs))
	rmse := make([]opts.LineData, 0, len(inputs))

	for _, in := range inputs {
		x = append(x, shortLabel(in))
		aic = append(aic, opts.LineData{Value: in.Analysis.AIC})
		bic = append(bic, opts.LineData{Value: in.Analysis.BIC})
		ks = append(ks, opts.LineData{Value: in.Analysis.KSStatistic})
		rmse = append(rmse, opts.LineData{Value: in.Analysis.RMSE})
	}

	line.SetXAxis(x).
		AddSeries("AIC", aic).
		AddSeries("BIC", bic).
		AddSeries("KS", ks).
		AddSeries("RMSE", rmse)

	return line
}

// BuildDistributionMetricsChart erstellt ein Metrics Chart für jede Verteilung.
func BuildDistributionMetricsChart(inputs []DistributionPlotInput) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Verteilungs Fit-Güte",
			Subtitle: "AIC, BIC, KS und RMSE je Serie",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Serie"}),
	)

	x := make([]string, 0, len(inputs))
	aic := make([]opts.LineData, 0, len(inputs))
	bic := make([]opts.LineData, 0, len(inputs))
	ks := make([]opts.LineData, 0, len(inputs))
	rmse := make([]opts.LineData, 0, len(inputs))

	for _, in := range inputs {
		x = append(x, DistributionShortLabel(in))
		aic = append(aic, opts.LineData{Value: in.Metrics.AIC})
		bic = append(bic, opts.LineData{Value: in.Metrics.BIC})
		ks = append(ks, opts.LineData{Value: in.Metrics.KSStatistic})
		rmse = append(rmse, opts.LineData{Value: in.Metrics.RMSE})
	}

	line.SetXAxis(x).
		AddSeries("AIC", aic).
		AddSeries("BIC", bic).
		AddSeries("KS", ks).
		AddSeries("RMSE", rmse)

	return line
}

func BuildWeibullComparisonChart(inputs []WeibullPlotInput) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Weibull Parametervergleich",
			Subtitle: "Shape, Scale und Location je Serie",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Serie"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Parameterwert"}),
	)

	x := make([]string, 0, len(inputs))
	shape := make([]opts.BarData, 0, len(inputs))
	scale := make([]opts.BarData, 0, len(inputs))
	location := make([]opts.BarData, 0, len(inputs))

	for _, in := range inputs {
		x = append(x, shortLabel(in))
		shape = append(shape, opts.BarData{Value: in.Analysis.Shape})
		scale = append(scale, opts.BarData{Value: in.Analysis.Scale})
		location = append(location, opts.BarData{Value: in.Analysis.Location})
	}

	bar.SetXAxis(x).
		AddSeries("Shape", shape).
		AddSeries("Scale", scale).
		AddSeries("Location", location)

	return bar
}

func BuildWeibullDistributionOverview(inputs []WeibullPlotInput) *components.Page {
	page := components.NewPage()
	page.PageTitle = "Weibull Overview"

	for _, in := range inputs {
		page.AddCharts(
			BuildWeibullHistogramPDFChart(in),
			BuildWeibullCDFChart(in),
		)
	}

	return page
}

func BuildWeibullPDFLine(in WeibullPlotInput) *charts.Line {
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    chartTitle("Weibull PDF", in),
			Subtitle: "Fitted dichtekurve",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true), Trigger: "axis"}),
		charts.WithLegendOpts(opts.Legend{Show: opts.Bool(false)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Windgeschwindigkeit [m/s]"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Dichte"}),
	)

	x, y := densitySeries(in.FittedPDF)
	line.SetXAxis(x).AddSeries("PDF", y)
	return line
}

func BuildWeibullComparisonHeatmap(comparisons []models.WeibullComparisonResult) *charts.HeatMap {
	heat := charts.NewHeatMap()
	heat.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Weibull Distanzmatrix",
			Subtitle: "KS-Statistik zwischen Serien",
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Serie"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Serie"}),
	)

	labels := uniqueComparisonLabels(comparisons)
	index := map[string]int{}
	for i, label := range labels {
		index[label] = i
	}

	data := make([]opts.HeatMapData, 0, len(comparisons))
	for _, c := range comparisons {
		left := index[comparisonLabel(c.LeftSeriesName, c.LeftHeightM)]
		right := index[comparisonLabel(c.RightSeriesName, c.RightHeightM)]
		data = append(data, opts.HeatMapData{
			Value: []interface{}{left, right, c.KSStatistic},
		})
	}

	heat.SetXAxis(labels)

	heat.AddSeries("KS", data)
	return heat
}

func chartTitle(base string, in WeibullPlotInput) string {
	if in.LocationName == "" && in.HeightM == 0 {
		return base
	}
	return strings.TrimSpace(fmt.Sprintf("%s – %s @ %dm", base, in.LocationName, in.HeightM))
}

func shortLabel(in WeibullPlotInput) string {
	if in.LocationName == "" && in.SeriesName == "" {
		return fmt.Sprintf("%dm", in.HeightM)
	}
	if in.SeriesName != "" {
		return in.SeriesName
	}
	return fmt.Sprintf("%s @ %dm", in.LocationName, in.HeightM)
}

func cdfSeries(points []statistics.CDFPoint) ([]string, []float64) {
	x := make([]string, 0, len(points))
	y := make([]float64, 0, len(points))
	for _, p := range points {
		x = append(x, fmt.Sprintf("%.4f", p.X))
		y = append(y, p.Y)
	}
	return x, y
}

func densitySeries(points []statistics.DensityPoint) ([]string, []opts.LineData) {
	x := make([]string, 0, len(points))
	y := make([]opts.LineData, 0, len(points))
	for _, p := range points {
		x = append(x, fmt.Sprintf("%.4f", p.X))
		y = append(y, opts.LineData{Value: p.Y})
	}
	return x, y
}

func toLineData(values []float64) []opts.LineData {
	out := make([]opts.LineData, 0, len(values))
	for _, v := range values {
		out = append(out, opts.LineData{Value: v})
	}
	return out
}

func mergeSortedXAxis(a, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	sort.Strings(out)
	out = uniqueStrings(out)
	return out
}

func alignSeries(xAxis, seriesX []string, seriesY []float64) []float64 {
	m := make(map[string]float64, len(seriesX))
	for i := range seriesX {
		m[seriesX[i]] = seriesY[i]
	}

	out := make([]float64, len(xAxis))
	for i, x := range xAxis {
		if v, ok := m[x]; ok {
			out[i] = v
			continue
		}
		out[i] = math.NaN()
	}
	return out
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	last := ""
	for i, s := range in {
		if i == 0 || s != last {
			out = append(out, s)
			last = s
		}
	}
	return out
}

func comparisonLabel(name string, height int) string {
	return fmt.Sprintf("%s@%dm", name, height)
}

func uniqueComparisonLabels(comparisons []models.WeibullComparisonResult) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(comparisons)*2)

	for _, c := range comparisons {
		left := comparisonLabel(c.LeftSeriesName, c.LeftHeightM)
		right := comparisonLabel(c.RightSeriesName, c.RightHeightM)

		if _, ok := seen[left]; !ok {
			seen[left] = struct{}{}
			out = append(out, left)
		}
		if _, ok := seen[right]; !ok {
			seen[right] = struct{}{}
			out = append(out, right)
		}
	}

	sort.Strings(out)
	return out
}

func normalizeSeries(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func histogramXAxisAndCenters(hist []statistics.Bin) ([]string, []float64) {
	x := make([]string, 0, len(hist))
	centers := make([]float64, 0, len(hist))
	for _, b := range hist {
		x = append(x, fmt.Sprintf("%.2f-%.2f", b.Min, b.Max))
		centers = append(centers, (b.Min+b.Max)/2.0)
	}
	return x, centers
}

func densityAtXVals(points []statistics.DensityPoint, xVals []float64) []float64 {
	out := make([]float64, len(xVals))
	if len(points) == 0 {
		return out
	}

	sorted := append([]statistics.DensityPoint(nil), points...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].X < sorted[j].X })

	for i, x := range xVals {
		if x <= sorted[0].X {
			out[i] = normalizeSeries(sorted[0].Y)
			continue
		}
		last := len(sorted) - 1
		if x >= sorted[last].X {
			out[i] = normalizeSeries(sorted[last].Y)
			continue
		}

		for j := 1; j < len(sorted); j++ {
			left, right := sorted[j-1], sorted[j]
			if x >= left.X && x <= right.X {
				dx := right.X - left.X
				if dx == 0 {
					out[i] = normalizeSeries(left.Y)
					break
				}
				t := (x - left.X) / dx
				out[i] = normalizeSeries(left.Y + t*(right.Y-left.Y))
				break
			}
		}
	}

	return out
}

func cdfAtUpperBounds(points []statistics.CDFPoint, xVals []float64) []float64 {
	out := make([]float64, len(xVals))
	if len(points) == 0 {
		return out
	}

	sorted := append([]statistics.CDFPoint(nil), points...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].X < sorted[j].X })

	idx := 0
	current := 0.0
	for i, x := range xVals {
		for idx < len(sorted) && sorted[idx].X <= x {
			current = normalizeSeries(sorted[idx].Y)
			idx++
		}
		if current < 0 {
			current = 0
		}
		if current > 1 {
			current = 1
		}
		out[i] = current
	}

	return out
}
