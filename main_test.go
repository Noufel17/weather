// main_test.go - Unit tests for the weather web server
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	expected := "Casablanca, Morocco: 22.0°C, Partly cloudy"
	result := FormatCurrentWeather(weather)
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// TestWeatherHandler tests the main HTTP handler
func TestWeatherHandler(t *testing.T) {
	t.Run("Successful request with city", func(t *testing.T) {
		// Mock the API response for this specific test case
		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				Response: &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"location": {"name": "TestCity", "country": "TestCountry"}, "current": {"temp_c": 25.0, "condition": {"text": "Sunny"}}}`)),
				},
			},
		}

		// Create a new App instance and inject the mock weather service
		app := &App{weatherService: NewWeatherService("test-key", mockClient)}
		req := httptest.NewRequest("GET", "/weather?city=TestCity", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.weatherHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		expected := `{"weather":"TestCity, TestCountry: 25.0°C, Sunny"}` + "\n"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %q want %q", rr.Body.String(), expected)
		}
	})

	t.Run("Request without city", func(t *testing.T) {
		// Mock the API response for this specific test case
		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				Response: &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"location": {"name": "Algiers", "country": "Algeria"}, "current": {"temp_c": 20.0, "condition": {"text": "Cloudy"}}}`)),
				},
			},
		}

		app := &App{weatherService: NewWeatherService("test-key", mockClient)}
		req := httptest.NewRequest("GET", "/weather", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.weatherHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		expected := `{"weather":"Algiers, Algeria: 20.0°C, Cloudy"}` + "\n"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %q want %q", rr.Body.String(), expected)
		}
	})

	t.Run("API error response", func(t *testing.T) {
		// Mock the API response to be a 401 error
		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				Response: &http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader("Unauthorized")),
				},
			},
		}

		app := &App{weatherService: NewWeatherService("test-key", mockClient)}
		req := httptest.NewRequest("GET", "/weather?city=TestCity", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.weatherHandler)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
		expected := "Error fetching weather: API returned status 401 for city TestCity\n"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %q want %q", rr.Body.String(), expected)
		}
	})

	t.Run("API key not set", func(t *testing.T) {
		// Mock the API response to be a 400 error because the key is missing
		mockClient := &http.Client{
			Transport: &mockRoundTripper{
				Response: &http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader("Bad Request")),
				},
			},
		}

		app := &App{
			weatherService: NewWeatherService("", mockClient),
		}

		req := httptest.NewRequest("GET", "/weather", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(app.weatherHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}

		expected := "Error fetching weather: API returned status 400 for city Algiers\n"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %q want %q", rr.Body.String(), expected)
		}
	})
}
