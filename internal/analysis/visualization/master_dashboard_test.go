package visualization

import (
	"testing"
	"time"

	"wind_analysis/internal/analysis/interpolation"
	"wind_analysis/models"
)

func TestNewMasterDashboardData(t *testing.T) {
	master := NewMasterDashboardData()
	
	if master == nil {
		t.Fatal("NewMasterDashboardData returned nil")
	}
	
	if master.Locations == nil {
		t.Error("Locations slice should be initialized")
	}
	
	if master.WindSpeedData == nil {
		t.Error("WindSpeedData map should be initialized")
	}
	
	if master.HellmannData == nil {
		t.Error("HellmannData map should be initialized")
	}
	
	if master.DistributionData == nil {
		t.Error("DistributionData map should be initialized")
	}
	
	if master.ValidationData == nil {
		t.Error("ValidationData map should be initialized")
	}
	
	if master.ErrorShapeData == nil {
		t.Error("ErrorShapeData map should be initialized")
	}
}

func TestAddLocation(t *testing.T) {
	master := NewMasterDashboardData()
	
	// Ersten Standort hinzufügen
	master.AddLocation("location1")
	
	if len(master.Locations) != 1 {
		t.Errorf("Expected 1 location, got %d", len(master.Locations))
	}
	
	if master.Locations[0] != "location1" {
		t.Errorf("Expected 'location1', got '%s'", master.Locations[0])
	}
	
	// Zweiten Standort hinzufügen
	master.AddLocation("location2")
	
	if len(master.Locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(master.Locations))
	}
	
	// Versuch, denselben Standort nochmal hinzuzufügen (sollte nicht dupliziert werden)
	master.AddLocation("location1")
	
	if len(master.Locations) != 2 {
		t.Errorf("Expected 2 locations (no duplicate), got %d", len(master.Locations))
	}
	
	// Prüfen ob sortiert
	master.AddLocation("location3")
	if master.Locations[0] != "location1" || master.Locations[1] != "location2" || master.Locations[2] != "location3" {
		t.Error("Locations should be sorted")
	}
}

func TestAddWindSpeedData(t *testing.T) {
	master := NewMasterDashboardData()
	
	points := []TimeSeriesPoint{
		{Time: time.Now(), Value: 5.5},
		{Time: time.Now().Add(time.Hour), Value: 6.0},
	}
	
	master.AddWindSpeedData("test_location", points)
	
	if !master.HasWindSpeedData("test_location") {
		t.Error("WindSpeedData should be present for test_location")
	}
	
	retrieved := master.GetWindSpeedTimeSeries("test_location")
	if len(retrieved.Points) != len(points) {
		t.Errorf("Expected %d points, got %d", len(points), len(retrieved.Points))
	}
	
	// Prüfen ob Standort automatisch zur Locations-Liste hinzugefügt wurde
	if !contains(master.Locations, "test_location") {
		t.Error("Location should be automatically added to Locations list")
	}
}

func TestAddHellmannData(t *testing.T) {
	master := NewMasterDashboardData()
	
	points := []TimeSeriesPoint{
		{Time: time.Now(), Value: 0.15},
		{Time: time.Now().Add(time.Hour), Value: 0.18},
	}
	
	master.AddHellmannData("test_location", points)
	
	if !master.HasHellmannData("test_location") {
		t.Error("HellmannData should be present for test_location")
	}
	
	retrieved := master.GetHellmannTimeSeries("test_location")
	if len(retrieved.Points) != len(points) {
		t.Errorf("Expected %d points, got %d", len(points), len(retrieved.Points))
	}
}

func TestAddDistributionData(t *testing.T) {
	master := NewMasterDashboardData()
	
	inputs := []DistributionPlotInput{
		{
			SeriesName:   "test_series",
			LocationName: "test_location",
			HeightM:      100,
		},
	}
	
	master.AddDistributionData("test_location", inputs)
	
	if !master.HasDistributionData("test_location") {
		t.Error("DistributionData should be present for test_location")
	}
	
	retrieved := master.GetDistributionInputs("test_location")
	if len(retrieved) != len(inputs) {
		t.Errorf("Expected %d inputs, got %d", len(inputs), len(retrieved))
	}
}

func TestAddValidationData(t *testing.T) {
	master := NewMasterDashboardData()
	
	rows := []MetricRow{
		{
			Location:    "test_location",
			HeightM:     100.0,
			MAE:         0.5,
			RMSE:        0.7,
			Correlation: 0.95,
			SampleCount: 1000,
		},
	}
	
	master.AddValidationData("test_location", rows)
	
	if !master.HasValidationData("test_location") {
		t.Error("ValidationData should be present for test_location")
	}
	
	retrieved := master.GetValidationRows("test_location")
	if len(retrieved) != len(rows) {
		t.Errorf("Expected %d rows, got %d", len(rows), len(retrieved))
	}
}

func TestAddErrorShapeData(t *testing.T) {
	master := NewMasterDashboardData()
	
	heightMap := map[float64]interpolation.ValidationResult{
		100.0: {
			MAE:         0.5,
			RMSE:        0.7,
			Correlation: 0.95,
			SampleCount: 1000,
		},
	}
	
	master.AddErrorShapeData("test_location", heightMap)
	
	if !master.HasErrorShapeData("test_location") {
		t.Error("ErrorShapeData should be present for test_location")
	}
	
	retrieved := master.GetErrorShapeData("test_location")
	if len(retrieved) != len(heightMap) {
		t.Errorf("Expected %d heights, got %d", len(heightMap), len(retrieved))
	}
}

func TestSetTimeRange(t *testing.T) {
	master := NewMasterDashboardData()
	
	start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	
	master.SetTimeRange(start, end)
	
	if !master.StartTime.Equal(start) {
		t.Errorf("Expected start time %v, got %v", start, master.StartTime)
	}
	
	if !master.EndTime.Equal(end) {
		t.Errorf("Expected end time %v, got %v", end, master.EndTime)
	}
}

func TestGetAllWindSpeedTimeSeries(t *testing.T) {
	master := NewMasterDashboardData()
	
	points1 := []TimeSeriesPoint{{Time: time.Now(), Value: 5.5}}
	points2 := []TimeSeriesPoint{{Time: time.Now(), Value: 6.0}}
	
	master.AddWindSpeedData("location1", points1)
	master.AddWindSpeedData("location2", points2)
	
	all := master.GetAllWindSpeedTimeSeries()
	
	if len(all) != 2 {
		t.Errorf("Expected 2 time series, got %d", len(all))
	}
}

func TestGetAllHellmannTimeSeries(t *testing.T) {
	master := NewMasterDashboardData()
	
	points1 := []TimeSeriesPoint{{Time: time.Now(), Value: 0.15}}
	points2 := []TimeSeriesPoint{{Time: time.Now(), Value: 0.18}}
	
	master.AddHellmannData("location1", points1)
	master.AddHellmannData("location2", points2)
	
	all := master.GetAllHellmannTimeSeries()
	
	if len(all) != 2 {
		t.Errorf("Expected 2 time series, got %d", len(all))
	}
}

func TestGetAllValidationRows(t *testing.T) {
	master := NewMasterDashboardData()
	
	rows1 := []MetricRow{{Location: "location1", HeightM: 100.0}}
	rows2 := []MetricRow{{Location: "location2", HeightM: 100.0}}
	
	master.AddValidationData("location1", rows1)
	master.AddValidationData("location2", rows2)
	
	all := master.GetAllValidationRows()
	
	if len(all) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(all))
	}
}

func TestMapLocationToMasterDashboard(t *testing.T) {
	location := models.Location{
		Name:      "test_location",
		Latitude:  52.0,
		Longitude: 10.0,
	}
	
	records := []models.WindRecord{
		{
			Time: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			WindData: models.WindData{
				WindSpeed_10m: func() *float64 { v := 5.5; return &v }(),
			},
		},
	}
	
	exponents := []interpolation.HellmannExponentResult{
		{
			Time:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			Alpha: 0.15,
		},
	}
	
	distributionInputs := []DistributionPlotInput{
		{
			SeriesName:   "test_series",
			LocationName: "test_location",
			HeightM:      100,
		},
	}
	
	master, err := MapLocationToMasterDashboard(
		location,
		records,
		exponents,
		nil, // validationByLocation
		distributionInputs,
	)
	
	if err != nil {
		t.Fatalf("MapLocationToMasterDashboard failed: %v", err)
	}
	
	if master == nil {
		t.Fatal("MapLocationToMasterDashboard returned nil")
	}
	
	if !contains(master.Locations, "test_location") {
		t.Error("Location should be in master dashboard")
	}
	
	if !master.HasWindSpeedData("test_location") {
		t.Error("WindSpeedData should be present")
	}
	
	if !master.HasHellmannData("test_location") {
		t.Error("HellmannData should be present")
	}
	
	if !master.HasDistributionData("test_location") {
		t.Error("DistributionData should be present")
	}
}

func TestBuildWindSpeedTimeSeriesForLocation(t *testing.T) {
	records := []models.WindRecord{
		{
			Time: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			WindData: models.WindData{
				WindSpeed_10m: func() *float64 { v := 5.5; return &v }(),
			},
		},
		{
			Time: time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
			WindData: models.WindData{
				WindSpeed_10m: func() *float64 { v := 6.0; return &v }(),
			},
		},
	}
	
	ts, err := buildWindSpeedTimeSeriesForLocation("test", records)
	
	if err != nil {
		t.Fatalf("buildWindSpeedTimeSeriesForLocation failed: %v", err)
	}
	
	if len(ts.Points) != 2 {
		t.Errorf("Expected 2 points, got %d", len(ts.Points))
	}
	
	if ts.LocationName != "test" {
		t.Errorf("Expected location name 'test', got '%s'", ts.LocationName)
	}
}

func TestExtractRecordTimeAndSpeed(t *testing.T) {
	record := models.WindRecord{
		Time: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		WindData: models.WindData{
			WindSpeed_10m: func() *float64 { v := 5.5; return &v }(),
		},
	}
	
	extractedTime, v, ok := extractRecordTimeAndSpeed(record)
	
	if !ok {
		t.Error("extractRecordTimeAndSpeed should succeed")
	}
	
	if v != 5.5 {
		t.Errorf("Expected wind speed 5.5, got %f", v)
	}
	
	expectedTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	if !extractedTime.Equal(expectedTime) {
		t.Errorf("Expected time %v, got %v", expectedTime, extractedTime)
	}
}

func TestExtractRecordTimeAndSpeedPriority(t *testing.T) {
	// Test: Priorität sollte 10m > 80m > 100m sein
	record := models.WindRecord{
		Time: time.Now(),
		WindData: models.WindData{
			WindSpeed_10m:  func() *float64 { v := 5.5; return &v }(),
			WindSpeed_80m:  func() *float64 { v := 8.0; return &v }(),
			WindSpeed_100m: func() *float64 { v := 10.0; return &v }(),
		},
	}
	
	_, v, ok := extractRecordTimeAndSpeed(record)
	
	if !ok {
		t.Error("extractRecordTimeAndSpeed should succeed")
	}
	
	if v != 5.5 {
		t.Errorf("Expected wind speed 5.5 (10m priority), got %f", v)
	}
}

func TestExtractRecordTimeAndSpeedFallback(t *testing.T) {
	// Test: Fallback auf 80m wenn 10m nicht verfügbar
	record := models.WindRecord{
		Time: time.Now(),
		WindData: models.WindData{
			WindSpeed_80m: func() *float64 { v := 8.0; return &v }(),
		},
	}
	
	_, v, ok := extractRecordTimeAndSpeed(record)
	
	if !ok {
		t.Error("extractRecordTimeAndSpeed should succeed with fallback")
	}
	
	if v != 8.0 {
		t.Errorf("Expected wind speed 8.0 (80m fallback), got %f", v)
	}
}
