// main.go - Refactored for testability
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

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

type WeatherService struct {
	APIKey string
	Client *http.Client
}

// NewWeatherService creates a new WeatherService instance
func NewWeatherService(apiKey string, client *http.Client) *WeatherService {
	return &WeatherService{
		APIKey: apiKey,
		Client: client,
	}
}

// FetchWeatherData fetches weather data from the API
func (ws *WeatherService) FetchWeatherData(city string) (*Weather, error) {
	apiUrl := fmt.Sprintf("https://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=1", ws.APIKey, city)

	resp, err := ws.Client.Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

// FormatCurrentWeather formats the current weather for printing
func FormatCurrentWeather(weather *Weather) string {
	location := fmt.Sprintf("%s, %s", weather.Location.Name, weather.Location.Country)
	temp := fmt.Sprintf("%.1f°C", weather.Current.TempC)
	condition := weather.Current.Condition.Text
	return fmt.Sprintf("%s: %s, %s", location, temp, condition)
}

// FormatHourlyForecast formats the hourly forecast data
func FormatHourlyForecast(weather *Weather) []string {
	var forecasts []string

	if len(weather.Forecast.Forecastday) == 0 {
		return forecasts
	}

	hours := weather.Forecast.Forecastday[0].Hour
	for _, hour := range hours {
		date := time.Unix(hour.TimeEpoch, 0)
		temp := fmt.Sprintf("%.1f°C", hour.TempC)
		condition := hour.Condition.Text
		rainChance := fmt.Sprintf("%.0f%%", hour.ChanceOfRain)

		forecast := fmt.Sprintf("%s - %s, %s - %s",
			date.Local().Format("15:04"),
			temp,
			rainChance,
			condition,
		)

		forecasts = append(forecasts, forecast)
	}

	return forecasts
}

// PrintForecast prints the forecast
func PrintForecast(forecasts []string, weather *Weather) {
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
			fmt.Println(forecasts[hourIndex])
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
		apiKey = "YOUR_API_KEY_HERE"
	}

	weatherService := NewWeatherService(apiKey, &http.Client{Timeout: 10 * time.Second})

	weather, err := weatherService.FetchWeatherData(city)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(FormatCurrentWeather(weather))

	forecasts := FormatHourlyForecast(weather)
	PrintForecast(forecasts, weather)
}
