package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"
)

// --- Configuration ---

// Coordinates for Copenhagen
const (
	weatherLat = 55.6761
	weatherLon = 12.5683
	// How long a cached result is "valid" before we fetch a new one
	weatherCacheTTL = 15 * time.Minute
)

// --- Cache ---

// weatherCache holds the most recently fetched weather result along with when it was fetched.
// sync.Mutex ensures that concurrent requests don't read/write the cache at the same time
// (relevant, since the Go server handles requests in parallel goroutines).
type weatherCacheEntry struct {
	mu        sync.Mutex // lock to protect data (mutual exclusion)
	data      map[string]interface{}
	fetchedAt time.Time
}

var weatherCacheStore = &weatherCacheEntry{}

// --- Open-Meteo's response structure (only the fields we use) ---

type openMeteoResponse struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		WindSpeed   float64 `json:"wind_speed_10m"`
		Humidity    float64 `json:"relative_humidity_2m"`
		WeatherCode int     `json:"weather_code"`
	} `json:"current"`
}

// fetchWeatherFromProvider fetches FRESH data from Open-Meteo.
// Only called when the cache is empty or stale.
func fetchWeatherFromProvider() (map[string]interface{}, error) {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,wind_speed_10m,relative_humidity_2m,weather_code&wind_speed_unit=ms",
		weatherLat, weatherLon,
	)

	client := http.Client{Timeout: 5 * time.Second} // Important: sets a timeout so a slow third-party response doesn't cause our own /api/weather to exceed the simulation's 6-second requirement
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not contact the weather service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather service responded with status code %d", resp.StatusCode)
	}

	// decodes the JSON response into the parsing struct openMeteoResponse, which only contains the fields we use
	var parsed openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("could not read response from weather service: %w", err)
	}

	return map[string]interface{}{
		"temperature":  parsed.Current.Temperature,
		"wind_speed":   parsed.Current.WindSpeed,
		"humidity":     parsed.Current.Humidity,
		"weather_code": parsed.Current.WeatherCode,
	}, nil
}

// getWeatherData returns cached data if it's fresh enough — otherwise fetches new data.
// This is the mechanism that solves the scaling question: no matter how many
// concurrent calls occur, only ONE of them (in practice) hits the external
// service, the rest get the cached result.
func getWeatherData() (map[string]interface{}, error) {
	weatherCacheStore.mu.Lock()
	defer weatherCacheStore.mu.Unlock()

	cacheIsStale := time.Since(weatherCacheStore.fetchedAt) > weatherCacheTTL
	if weatherCacheStore.data == nil || cacheIsStale {
		fresh, err := fetchWeatherFromProvider()
		if err != nil {
			// If we HAVE an old cached result, it's better to serve it
			// (slightly stale weather) than to fail entirely if the third-party service is down.
			if weatherCacheStore.data != nil {
				return weatherCacheStore.data, nil
			}
			return nil, err
		}
		weatherCacheStore.data = fresh
		weatherCacheStore.fetchedAt = time.Now()
	}

	return weatherCacheStore.data, nil
}

// --- JSON API endpoint ---

// @Summary Weather
// @Success 200 {object} StandardResponse
// @Failure 502 {object} HTTPValidationError
// @Router /api/weather [get]
func apiWeatherHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data, err := getWeatherData()
	if err != nil {
		w.WriteHeader(http.StatusBadGateway) // 502: we couldn't get data from a THIRD PARTY, not our own error
		json.NewEncoder(w).Encode(HTTPValidationError{
			Detail: []ValidationError{
				{Loc: []interface{}{"weather"}, Msg: "Could not retrieve weather data", Type: "upstream_error"},
			},
		})
		return
	}

	json.NewEncoder(w).Encode(StandardResponse{Data: data})
}

// --- HTML page ---

// init registers the weather template alongside the existing ones in the shared
// `pages` map (declared in main.go). Package-level variables are guaranteed to be
// initialized before any init() function runs, so `pages` already exists by the
// time this runs, regardless of file order.
func init() {
	pages["weather"] = template.Must(template.ParseFiles(htmlDir+"layout.html", htmlDir+"weather.html"))
}

// @Summary Serve Weather Page
// @Router /weather [get]
func serveWeatherPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	raw, err := getWeatherData()
	page := PageData{Title: "Weather"}

	if err != nil {
		page.Error = "Could not retrieve the weather forecast right now. Please try again shortly."
	} else {
		info := &WeatherInfo{}
		if temp, ok := raw["temperature"].(float64); ok {
			info.Temperature = temp
		}
		if wind, ok := raw["wind_speed"].(float64); ok {
			info.WindSpeed = wind
		}
		if hum, ok := raw["humidity"].(float64); ok {
			info.Humidity = hum
		}
		page.Weather = info
	}

	pages["weather"].ExecuteTemplate(w, "layout", page)
}
