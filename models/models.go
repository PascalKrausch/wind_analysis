package models

import "time"

// -----------------------------------------------------------------------------
// 1. API & Core Domain Models (Bestehend)
// -----------------------------------------------------------------------------

type WindData struct {
	WindSpeed_10m  *float64 `json:"wind_speed_10m"`
	WindSpeed_80m  *float64 `json:"wind_speed_80m"`
	WindSpeed_100m *float64 `json:"wind_speed_100m"`
	WindSpeed_120m *float64 `json:"wind_speed_120m"`
	WindSpeed_180m *float64 `json:"wind_speed_180m"`
	WindSpeed_200m *float64 `json:"wind_speed_200m"`

	WindDirection_10m  *float64 `json:"wind_direction_10m"`
	WindDirection_80m  *float64 `json:"wind_direction_80m"`
	WindDirection_100m *float64 `json:"wind_direction_100m"`
	WindDirection_120m *float64 `json:"wind_direction_120m"`
	WindDirection_180m *float64 `json:"wind_direction_180m"`
	WindDirection_200m *float64 `json:"wind_direction_200m"`
}

type Location struct {
	ID        int64   `json:"id,omitempty"`
	Name      string  `json:"name" yaml:"name"`
	Latitude  float64 `json:"latitude" yaml:"latitude"`
	Longitude float64 `json:"longitude" yaml:"longitude"`
}

type WindRecord struct {
	Time     time.Time `json:"time"`
	Location Location  `json:"location"`
	WindData WindData  `json:"wind_data"`
}

type OpenMeteoResponse struct {
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Timezone  string     `json:"timezone"`
	Hourly    HourlyData `json:"hourly"`
}

type HourlyData struct {
	Time               []string  `json:"time"`
	WindSpeed_10m      []float64 `json:"wind_speed_10m"`
	WindSpeed_80m      []float64 `json:"wind_speed_80m"`
	WindSpeed_100m     []float64 `json:"wind_speed_100m,omitempty"`
	WindSpeed_120m     []float64 `json:"wind_speed_120m"`
	WindSpeed_180m     []float64 `json:"wind_speed_180m"`
	WindSpeed_200m     []float64 `json:"wind_speed_200m,omitempty"`
	WindDirection_10m  []float64 `json:"wind_direction_10m"`
	WindDirection_80m  []float64 `json:"wind_direction_80m"`
	WindDirection_100m []float64 `json:"wind_direction_100m,omitempty"`
	WindDirection_120m []float64 `json:"wind_direction_120m"`
	WindDirection_180m []float64 `json:"wind_direction_180m"`
	WindDirection_200m []float64 `json:"wind_direction_200m,omitempty"`
}

// -----------------------------------------------------------------------------
// 2. Pipeline & Worker Architecture Models (Neu)
// -----------------------------------------------------------------------------

// FetchTask repräsentiert eine atomare Arbeitseinheit für die Worker-Pipeline
type FetchTask struct {
	ID           string    `json:"id"`
	LocationName string    `json:"location_name"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	RetryCount   int       `json:"retry_count"`
}

// TaskResult dient dem Monitoring und Logging der verarbeiteten Jobs
type TaskResult struct {
	Task         FetchTask `json:"task"`
	RecordCount  int       `json:"record_count"`
	DurationMs   int64     `json:"duration_ms"`
	Err          error     `json:"-"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// Config hält die Parameter für Raster-Generierung und Worker-Pipeline
type Config struct {
	LocationList []Location `yaml:"locationlist"`

	Timeframe struct {
		Start      string `yaml:"start"`
		End        string `yaml:"end"`
		ChunkYears int    `yaml:"chunk_years"`
	} `yaml:"timeframe"`
	Pipeline struct {
		Concurrency  int `yaml:"concurrency"`
		BatchSize    int `yaml:"batch_size"`
		RateLimitRPS int `yaml:"rate_limit_rps"`
	} `yaml:"pipeline"`
}

// -----------------------------------------------------------------------------
// 3. Weibull-Analysemodelle (Neu)
// -----------------------------------------------------------------------------

type WindSeriesDescriptor struct {
	LocationName string `json:"location_name" yaml:"location_name"`
	HeightM      int    `json:"height_m" yaml:"height_m"`
}

type WeibullAnalysisResult struct {
	SeriesName    string  `json:"series_name"`
	HeightM       int     `json:"height_m"`
	SampleSize    int     `json:"sample_size"`
	Shape         float64 `json:"shape"`
	Scale         float64 `json:"scale"`
	Location      float64 `json:"location"`
	LogLikelihood float64 `json:"log_likelihood"`
	AIC           float64 `json:"aic"`
	BIC           float64 `json:"bic"`
	KSStatistic   float64 `json:"ks_statistic"`
	RMSE          float64 `json:"rmse"`
}

type WeibullComparisonResult struct {
	LeftSeriesName  string  `json:"left_series_name"`
	RightSeriesName string  `json:"right_series_name"`
	LeftHeightM     int     `json:"left_height_m"`
	RightHeightM    int     `json:"right_height_m"`
	ShapeDelta      float64 `json:"shape_delta"`
	ScaleDelta      float64 `json:"scale_delta"`
	LocationDelta   float64 `json:"location_delta"`
	KSStatistic     float64 `json:"ks_statistic"`
	MeanAbsCDFDiff  float64 `json:"mean_abs_cdf_diff"`
}
