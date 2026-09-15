package visualization

import (
	"bytes"
	"os"
	"testing"

	"github.com/go-echarts/go-echarts/v2/opts"
)

func TestDefaultChartConfig(t *testing.T) {
	config := DefaultChartConfig()

	if config.Title != "Chart" {
		t.Errorf("Expected default title 'Chart', got '%s'", config.Title)
	}
	if config.XAxisName != "X" {
		t.Errorf("Expected default X axis name 'X', got '%s'", config.XAxisName)
	}
	if config.YAxisName != "Y" {
		t.Errorf("Expected default Y axis name 'Y', got '%s'", config.YAxisName)
	}
	if config.Width != "900px" {
		t.Errorf("Expected default width '900px', got '%s'", config.Width)
	}
	if config.Height != "500px" {
		t.Errorf("Expected default height '500px', got '%s'", config.Height)
	}
	// Theme should be set to a valid theme value
	if config.Theme == "" {
		t.Error("Expected theme to be set")
	}
}

func TestCreateLineChart(t *testing.T) {
	config := ChartConfig{
		Title:     "Test Line Chart",
		Subtitle:  "Test Subtitle",
		XAxisName: "Time",
		YAxisName: "Value",
	}

	line := CreateLineChart(config)

	if line == nil {
		t.Error("Expected non-nil line chart")
	}
}

func TestCreateScatterChart(t *testing.T) {
	config := ChartConfig{
		Title:     "Test Scatter Chart",
		XAxisName: "X Values",
		YAxisName: "Y Values",
	}

	scatter := CreateScatterChart(config)

	if scatter == nil {
		t.Error("Expected non-nil scatter chart")
	}
}

func TestCreateBarChart(t *testing.T) {
	config := ChartConfig{
		Title:     "Test Bar Chart",
		XAxisName: "Category",
		YAxisName: "Count",
	}

	bar := CreateBarChart(config)

	if bar == nil {
		t.Error("Expected non-nil bar chart")
	}
}

func TestCreateHeatMap(t *testing.T) {
	config := ChartConfig{
		Title:     "Test HeatMap",
		XAxisName: "X",
		YAxisName: "Y",
	}

	heatMap := CreateHeatMap(config)

	if heatMap == nil {
		t.Error("Expected non-nil heatmap")
	}
}

func TestCreateBoxPlot(t *testing.T) {
	config := ChartConfig{
		Title:     "Test BoxPlot",
		XAxisName: "Category",
		YAxisName: "Value",
	}

	boxPlot := CreateBoxPlot(config)

	if boxPlot == nil {
		t.Error("Expected non-nil boxplot")
	}
}

func TestRenderToFile(t *testing.T) {
	config := DefaultChartConfig()
	line := CreateLineChart(config)

	// Add some dummy data
	xData := []string{"A", "B", "C"}
	yData := []float64{1.0, 2.0, 3.0}
	AddLineSeries(line, "Test Series", xData, yData)

	// Create temporary file
	tmpFile := "/tmp/test_chart.html"
	defer os.Remove(tmpFile)

	err := RenderToFile(line, tmpFile)
	if err != nil {
		t.Errorf("Failed to render chart to file: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Expected file to be created")
	}
}

func TestRenderToWriter(t *testing.T) {
	config := DefaultChartConfig()
	line := CreateLineChart(config)

	// Add some dummy data
	xData := []string{"A", "B", "C"}
	yData := []float64{1.0, 2.0, 3.0}
	AddLineSeries(line, "Test Series", xData, yData)

	var buf bytes.Buffer
	err := RenderToWriter(line, &buf)
	if err != nil {
		t.Errorf("Failed to render chart to writer: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestAddLineSeries(t *testing.T) {
	config := DefaultChartConfig()
	line := CreateLineChart(config)

	xData := []string{"A", "B", "C"}
	yData := []float64{1.0, 2.0, 3.0}

	AddLineSeries(line, "Test Series", xData, yData)

	// If no error is thrown, the test passes
}

func TestAddScatterSeries(t *testing.T) {
	config := DefaultChartConfig()
	scatter := CreateScatterChart(config)

	data := []opts.ScatterData{
		{Value: []interface{}{1.0, 2.0}},
		{Value: []interface{}{2.0, 3.0}},
		{Value: []interface{}{3.0, 4.0}},
	}

	AddScatterSeries(scatter, "Test Series", data)

	// If no error is thrown, the test passes
}

func TestAddBarSeries(t *testing.T) {
	config := DefaultChartConfig()
	bar := CreateBarChart(config)

	xData := []string{"A", "B", "C"}
	yData := []float64{1.0, 2.0, 3.0}

	AddBarSeries(bar, "Test Series", xData, yData)

	// If no error is thrown, the test passes
}

func TestConvertToScatterData(t *testing.T) {
	xData := []float64{1.0, 2.0, 3.0}
	yData := []float64{2.0, 4.0, 6.0}

	result := ConvertToScatterData(xData, yData)

	if len(result) != 3 {
		t.Errorf("Expected 3 data points, got %d", len(result))
	}

	// Check first data point
	if result[0].Value.([]interface{})[0] != 1.0 {
		t.Errorf("Expected first X value 1.0, got %v", result[0].Value.([]interface{})[0])
	}
	if result[0].Value.([]interface{})[1] != 2.0 {
		t.Errorf("Expected first Y value 2.0, got %v", result[0].Value.([]interface{})[1])
	}
}

func TestConvertToScatterDataMismatchedLengths(t *testing.T) {
	xData := []float64{1.0, 2.0}
	yData := []float64{2.0, 4.0, 6.0}

	result := ConvertToScatterData(xData, yData)

	if result != nil {
		t.Error("Expected nil for mismatched lengths")
	}
}

func TestSetCommonSeriesOptions(t *testing.T) {
	config := DefaultChartConfig()
	line := CreateLineChart(config)

	// This should not panic
	SetCommonSeriesOptions(line)

	// If no error is thrown, the test passes
}
