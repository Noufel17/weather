// main.go - Refactored for testability
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/enescakir/emoji"
	"github.com/fatih/color"
)

// Weather struct to hold the API response data
type Weather struct {
	Location struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		Condition struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
		} `json:"condition"`
	} `json:"current"`
	Forecast struct {
		Forecastday []struct {
			Hour []struct {
				TimeEpoch    int64   `json:"time_epoch"`
				TempC        float64 `json:"temp_c"`
				Condition    struct {
					Text string `json:"text"`
					Icon string `json:"icon"`
				} `json:"condition"`
				ChanceOfRain float64 `json:"chance_of_rain"`
			} `json:"hour"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

// WeatherService encapsulates the weather API logic
type WeatherService struct {
	APIKey string
	Client *http.Client // Now a public field to allow for mocking
}

// Map of weather conditions to emojis
var emojis = map[string]emoji.Emoji{
	"Clear":              emoji.Sun,
	"Sunny":              emoji.Sun,
	"Patchy rain":        emoji.CloudWithRain,
	"Partly cloudy":      emoji.SunBehindCloud,
	"Cloudy":             emoji.Cloud,
	"Patchy rain nearby": emoji.CloudWithRain,
	"Rainy":              emoji.CloudWithRain,
}

// NewWeatherService creates a new weather service instance with a custom HTTP client.
// This allows us to use a mock client for testing.
func NewWeatherService(apiKey string, client *http.Client) *WeatherService {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &WeatherService{
		APIKey: apiKey,
		Client: client,
	}
}

// IsSimpleCondition checks if weather condition is in our emoji map
func IsSimpleCondition(category string) bool {
	switch category {
	case "Sunny", "Patchy rain", "Partly cloudy", "Cloudy", "Rainy", "Patchy rain nearby", "Clear":
		return true
	}
	return false
}

// GetWeatherEmoji returns appropriate emoji for weather condition
func GetWeatherEmoji(condition string) emoji.Emoji {
	condition = strings.TrimSpace(condition)
	if IsSimpleCondition(condition) {
		return emojis[condition]
	}
	return emoji.Cloud
}

// FetchWeatherData fetches weather data from API
func (ws *WeatherService) FetchWeatherData(city string) (*Weather, error) {
	url := fmt.Sprintf("https://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=1&aqi=no&alerts=no", ws.APIKey, city)

	resp, err := ws.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch weather data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d for city %s", resp.StatusCode, city)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var weather Weather
	if err := json.Unmarshal(body, &weather); err != nil {
		return nil, fmt.Errorf("failed to parse weather data: %w", err)
	}

	return &weather, nil
}

// FormatCurrentWeather formats current weather information
func FormatCurrentWeather(weather *Weather) string {
	location := weather.Location
	current := weather.Current
	condition := strings.TrimSpace(current.Condition.Text)
	emoji := GetWeatherEmoji(condition)

	return fmt.Sprintf("%s, %s: %.0f°C, %s %s",
		location.Name,
		location.Country,
		current.TempC,
		condition,
		emoji,
	)
}

// FormatHourlyForecast formats hourly forecast with colors for rain
func FormatHourlyForecast(weather *Weather) []string {
	if len(weather.Forecast.Forecastday) == 0 {
		return []string{}
	}

	hours := weather.Forecast.Forecastday[0].Hour
	var forecasts []string
	currentTime := time.Now()

	for _, hour := range hours {
		date := time.Unix(hour.TimeEpoch, 0)
		// Skip hours that are before the current hour
		if date.Before(currentTime) {
			continue
		}

		condition := strings.TrimSpace(hour.Condition.Text)
		emoji := GetWeatherEmoji(condition)

		forecast := fmt.Sprintf("%s - %.0f°C, %.0f%%, %s %s",
			date.Local().Format("15:04"),
			hour.TempC,
			hour.ChanceOfRain,
			condition,
			emoji,
		)

		forecasts = append(forecasts, forecast)
	}

	return forecasts
}

// PrintForecast prints the forecast to a given writer, with colors
func PrintForecast(w io.Writer, forecasts []string, weather *Weather) {
	if len(weather.Forecast.Forecastday) == 0 {
		return
	}

	hours := weather.Forecast.Forecastday[0].Hour
	currentTime := time.Now()
	hourIndex := 0

	for _, hour := range hours {
		date := time.Unix(hour.TimeEpoch, 0)
		if date.Before(currentTime) {
			continue
		}

		if hourIndex < len(forecasts) {
			if hour.ChanceOfRain >= 50 {
				color.New(color.FgRed).Fprintln(w, forecasts[hourIndex])
			} else {
				fmt.Fprintln(w, forecasts[hourIndex])
			}
			hourIndex++
		}
	}
}

// GetCityFromArgs extracts city name from command line arguments
func GetCityFromArgs(args []string) string {
	if len(args) >= 2 {
		return args[1]
	}
	return "Algiers" // default city
}

func main() {
	city := GetCityFromArgs(os.Args)

	// Use environment variable for API key, fallback to hardcoded for demo
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		apiKey = "94474d04349f43008d395834240102"
	}

	// Pass a nil client, which will be initialized to the default http.Client
	ws := NewWeatherService(apiKey, nil)

	weather, err := ws.FetchWeatherData(city)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print current weather
	fmt.Println(FormatCurrentWeather(weather))
	fmt.Println()

	// Print hourly forecast
	forecasts := FormatHourlyForecast(weather)
	PrintForecast(os.Stdout, forecasts, weather)
}