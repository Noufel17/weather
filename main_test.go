package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/enescakir/emoji"
)

// mockRoundTripper allows us to mock the HTTP client's behavior
type mockRoundTripper struct {
	Response     *http.Response
	Error        error
	RequestCheck func(*http.Request)
}

func (mrt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if mrt.RequestCheck != nil {
		mrt.RequestCheck(req)
	}
	return mrt.Response, mrt.Error
}

// TestNewWeatherService tests the creation of a new WeatherService
func TestNewWeatherService(t *testing.T) {
	apiKey := "test-key"
	client := &http.Client{}
	service := NewWeatherService(apiKey, client)

	if service.APIKey != apiKey {
		t.Errorf("Expected APIKey to be %s, got %s", apiKey, service.APIKey)
	}
	if service.Client != client {
		t.Errorf("Expected Client to be the one provided")
	}
}

// TestIsSimpleCondition uses a table-driven approach to test various conditions
func TestIsSimpleCondition(t *testing.T) {
	testCases := []struct {
		condition string
		expected  bool
	}{
		{"Sunny", true},
		{"Cloudy", true},
		{"Patchy rain", true},
		{"Non-simple condition", false},
	}

	for _, tc := range testCases {
		t.Run(tc.condition, func(t *testing.T) {
			result := IsSimpleCondition(tc.condition)
			if result != tc.expected {
				t.Errorf("For condition %q, expected %v, but got %v", tc.condition, tc.expected, result)
			}
		})
	}
}

// TestGetWeatherEmoji tests the emoji mapping
func TestGetWeatherEmoji(t *testing.T) {
	testCases := []struct {
		condition string
		expected  emoji.Emoji
	}{
		{"Sunny", emoji.Sun},
		{"Cloudy", emoji.Cloud},
		{"Non-simple condition", emoji.Cloud},
		{"", emoji.Cloud},
	}

	for _, tc := range testCases {
		t.Run(tc.condition, func(t *testing.T) {
			result := GetWeatherEmoji(tc.condition)
			if result != tc.expected {
				t.Errorf("For condition %q, expected %v, but got %v", tc.condition, tc.expected, result)
			}
		})
	}
}

// TestFetchWeatherData tests the API call with a mock client
func TestFetchWeatherData(t *testing.T) {
	// A sample successful API response body
	successBody := `{
		"location": {"name": "TestCity", "country": "TestCountry"},
		"current": {"temp_c": 25.5, "condition": {"text": "Sunny"}},
		"forecast": {"forecastday": [{"hour": []}]}
	}`

	// Test case for a successful API call
	t.Run("Success", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				Response: &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(successBody)),
				},
			},
		}

		ws := NewWeatherService("test-key", mockClient)
		weather, err := ws.FetchWeatherData("TestCity")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if weather.Location.Name != "TestCity" {
			t.Errorf("Expected city to be TestCity, got %s", weather.Location.Name)
		}
	})

	// Test case for a non-200 status code from the API
	t.Run("API Error", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				Response: &http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader("Bad Request")),
				},
			},
		}

		ws := NewWeatherService("test-key", mockClient)
		_, err := ws.FetchWeatherData("InvalidCity")
		if err == nil {
			t.Fatal("Expected an error, got none")
		}
		expectedErr := "API returned status 400 for city InvalidCity"
		if err.Error() != expectedErr {
			t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
		}
	})

	// Test case for invalid JSON response
	t.Run("Invalid JSON", func(t *testing.T) {
		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				Response: &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("invalid json")),
				},
			},
		}

		ws := NewWeatherService("test-key", mockClient)
		_, err := ws.FetchWeatherData("TestCity")
		if err == nil {
			t.Fatal("Expected an error, got none")
		}
		if !strings.Contains(err.Error(), "failed to parse weather data") {
			t.Errorf("Expected parse error, got %v", err)
		}
	})
}

// TestFormatCurrentWeather tests the formatting of current weather data
func TestFormatCurrentWeather(t *testing.T) {
	weather := &Weather{
		Location: struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		}{Name: "Casablanca", Country: "Morocco"},
		Current: struct {
			TempC     float64 `json:"temp_c"`
			Condition struct {
				Text string `json:"text"`
				Icon string `json:"icon"`
			} `json:"condition"`
		}{TempC: 22.0, Condition: struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
		}{Text: "Partly cloudy"}},
	}
	expected := fmt.Sprintf("Casablanca, Morocco: 22°C, Partly cloudy %v", emoji.SunBehindCloud)
	result := FormatCurrentWeather(weather)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestFormatHourlyForecast tests the formatting of the hourly forecast
func TestFormatHourlyForecast(t *testing.T) {
	now := time.Now()
	weather := &Weather{
		Forecast: struct {
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
		}{
			Forecastday: []struct {
				Hour []struct {
					TimeEpoch    int64   `json:"time_epoch"`
					TempC        float64 `json:"temp_c"`
					Condition    struct {
						Text string `json:"text"`
						Icon string `json:"icon"`
					} `json:"condition"`
					ChanceOfRain float64 `json:"chance_of_rain"`
				} `json:"hour"`
			}{
				{
					Hour: []struct {
						TimeEpoch    int64   `json:"time_epoch"`
						TempC        float64 `json:"temp_c"`
						Condition    struct {
							Text string `json:"text"`
							Icon string `json:"icon"`
						} `json:"condition"`
						ChanceOfRain float64 `json:"chance_of_rain"`
					}{
						{TimeEpoch: now.Add(-1 * time.Hour).Unix(), TempC: 15.0, Condition: struct {
							Text string `json:"text"`
							Icon string `json:"icon"`
						}{Text: "Cloudy"}, ChanceOfRain: 10},
						{TimeEpoch: now.Add(1 * time.Hour).Unix(), TempC: 18.0, Condition: struct {
							Text string `json:"text"`
							Icon string `json:"icon"`
						}{Text: "Sunny"}, ChanceOfRain: 5},
					},
				},
			},
		},
	}

	result := FormatHourlyForecast(weather)
	if len(result) != 1 {
		t.Fatalf("Expected 1 forecast, got %d", len(result))
	}

	expectedPrefix := now.Add(1 * time.Hour).Local().Format("15:04")
	if !strings.HasPrefix(result[0], expectedPrefix) {
		t.Errorf("Expected forecast to start with %q, but got %q", expectedPrefix, result[0])
	}
}

// TestGetCityFromArgs tests argument parsing
func TestGetCityFromArgs(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{"With city", []string{"cmd", "London"}, "London"},
		{"Without city", []string{"cmd"}, "Algiers"},
		{"Empty args", []string{}, "Algiers"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetCityFromArgs(tc.args)
			if result != tc.expected {
				t.Errorf("Expected %q, but got %q", tc.expected, result)
			}
		})
	}
}

// TestPrintForecast captures stdout to test the output
func TestPrintForecast(t *testing.T) {
	// Create a buffer to capture the output
	var buf bytes.Buffer

	forecasts := []string{
		"10:00 - 15°C, 10% - Sunny",
		"11:00 - 16°C, 60% - Rainy",
	}

	weather := &Weather{
		Forecast: struct {
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
		}{
			Forecastday: []struct {
				Hour []struct {
					TimeEpoch    int64   `json:"time_epoch"`
					TempC        float64 `json:"temp_c"`
					Condition    struct {
						Text string `json:"text"`
						Icon string `json:"icon"`
					} `json:"condition"`
					ChanceOfRain float64 `json:"chance_of_rain"`
				} `json:"hour"`
			}{
				{
					Hour: []struct {
						TimeEpoch    int64   `json:"time_epoch"`
						TempC        float64 `json:"temp_c"`
						Condition    struct {
							Text string `json:"text"`
							Icon string `json:"icon"`
						} `json:"condition"`
						ChanceOfRain float64 `json:"chance_of_rain"`
					}{
						{TimeEpoch: time.Now().Add(1 * time.Hour).Unix(), TempC: 15.0, Condition: struct {
							Text string `json:"text"`
							Icon string `json:"icon"`
						}{Text: "Sunny"}, ChanceOfRain: 10},
						{TimeEpoch: time.Now().Add(2 * time.Hour).Unix(), TempC: 16.0, Condition: struct {
							Text string `json:"text"`
							Icon string `json:"icon"`
						}{Text: "Rainy"}, ChanceOfRain: 60},
					},
				},
			},
		},
	}
	// Pass the buffer to the function to capture its output
	PrintForecast(&buf, forecasts, weather)
	output := buf.String()

	// The first line should not have color codes.
	if !strings.Contains(output, "10:00 - 15°C, 10% - Sunny\n") {
		t.Errorf("Expected output to contain '10:00 - 15°C, 10%% - Sunny\\n', but got %q", output)
	}

	// The second line with rain should have color codes.
	// We'll check for the full string including the ANSI escape codes.
	expectedColoredString := "\x1b[31m11:00 - 16°C, 60% - Rainy\x1b[0m\n"
	if !strings.Contains(output, expectedColoredString) {
		t.Errorf("Expected output to contain %q, but got %q", expectedColoredString, output)
	}
}