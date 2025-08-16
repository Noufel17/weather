package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// App struct holds the dependencies for the application
type App struct {
	weatherService *WeatherService
}

// Weather represents the JSON structure from the API
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

// WeatherService struct holds the API key and HTTP client
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

// FormatCurrentWeather formats the current weather data as a string
func FormatCurrentWeather(weather *Weather) string {
	location := fmt.Sprintf("%s, %s", weather.Location.Name, weather.Location.Country)
	temp := fmt.Sprintf("%.1f°C", weather.Current.TempC)
	condition := weather.Current.Condition.Text
	return fmt.Sprintf("%s: %s, %s", location, temp, condition)
}

// weatherHandler handles HTTP requests for weather data and is a method of App
func (a *App) weatherHandler(w http.ResponseWriter, r *http.Request) {
	// Get the city from the URL query parameters
	city := r.URL.Query().Get("city")
	if city == "" {
		city = "Algiers" // default city if none is provided
	}

	// Use the pre-initialized weather service
	weather, err := a.weatherService.FetchWeatherData(city)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching weather: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the formatted string as a JSON response
	formattedWeather := FormatCurrentWeather(weather)
	response := map[string]string{"weather": formattedWeather}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// main function to start the server
func main() {
	// Get API key from environment variable
	apiKey := "94474d04349f43008d395834240102"
	if apiKey == "" {
		log.Fatalf("API key not found. Please set the WEATHER_API_KEY environment variable.")
	}

	// Create a new App instance with a real HTTP client
	app := &App{
		weatherService: NewWeatherService(apiKey, &http.Client{Timeout: 10 * time.Second}),
	}

	// Register the handler for the /weather endpoint
	http.HandleFunc("/weather", app.weatherHandler)

	// Start the server on port 8080
	port := "8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
