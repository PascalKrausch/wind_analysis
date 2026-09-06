# Usage Example for Open-Meteo Client

This example demonstrates how to use the Open-Meteo client to fetch wind data with different height profiles.

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "wind_analysis/client"
    "wind_analysis/models"
)

func main() {
    // Create API client
    apiClient := client.NewOpenMeteoClient()

    // Define location (Berlin)
    lat := 52.52
    lon := 13.405
    locationName := "Berlin"

    // Define time range
    startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

    // Fetch wind data with automatic fallback for different height profiles
    response, err := apiClient.FetchWindDataWithFallback(lat, lon, startDate, endDate)
    if err != nil {
        log.Fatalf("Error fetching wind data: %v", err)
    }

    // Validate which heights are actually available
    availableHeights, err := client.ValidateWindData(response)
    if err != nil {
        log.Fatalf("Error validating wind data: %v", err)
    }

    fmt.Printf("Available heights: %v\n", availableHeights)
    fmt.Printf("Number of data points: %d\n", len(response.Hourly.Time))

    // Convert to LocationData for database storage
    locationData, err := client.ConvertToLocationData(response, locationName)
    if err != nil {
        log.Fatalf("Error converting data: %v", err)
    }

    fmt.Printf("Converted LocationData: %s with %d time points and %d parameters\n",
        locationData.Name, len(locationData.Times), len(locationData.Parameters))
}
```

## Different Height Profiles

The client automatically handles different height profiles based on the date:

### Before 2022 (Historical Data)
- Wind Speed: 10m, 100m
- Wind Direction: 10m, 100m

### From 2022 onwards (Current Data)
- Wind Speed: 10m, 80m, 100m, 120m, 180m, 200m
- Wind Direction: 10m, 80m, 100m, 120m, 180m, 200m

## API Response Structure

The API response includes the following fields:

```go
type OpenMeteoResponse struct {
    Latitude  float64
    Longitude float64
    Timezone  string
    Hourly    HourlyData
}

type HourlyData struct {
    Time                 []string
    WindSpeed_10m        []float64
    WindSpeed_80m        []float64
    WindSpeed_100m       []float64  // Optional
    WindSpeed_120m       []float64
    WindSpeed_180m       []float64
    WindSpeed_200m       []float64  // Optional
    WindDirection_10m    []float64
    WindDirection_80m    []float64
    WindDirection_100m   []float64  // Optional
    WindDirection_120m   []float64
    WindDirection_180m   []float64
    WindDirection_200m   []float64  // Optional
}
```

## Database Integration

The `ConvertToLocationData` function converts the API response into a format suitable for database storage:

```go
locationData, err := client.ConvertToLocationData(response, "Berlin")
// locationData.Parameters contains:
// - wind_speed_10m, wind_speed_80m, etc.
// - wind_direction_10m, wind_direction_80m, etc.
```

## Fallback Strategy

The client uses a fallback strategy:
1. First tries to fetch with the full height profile
2. If that fails, falls back to a reduced profile (10m, 100m)
3. This ensures compatibility with older data while taking advantage of newer detailed profiles when available
