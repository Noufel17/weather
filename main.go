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

type Weather struct {
	Location struct {
		Name string `json:"name"`
		Country string `json:"country"`
	}`json:"location"`
	Current struct {
		TempC float64 `json:"temp_c"`
		Condition struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
		}`json:"condition"`
	}`json:"current"` 
	Forecast struct {
		Forecastday []struct {
			Hour []struct {
				TimeEpoch int64 `json:"time_epoch"`
				TempC float64 `json:"temp_c"`
				Condition struct {
					Text string `json:"text"`
					Icon string `json:"icon"`
				}
				ChanceOfRain float64 `json:"chance_of_rain"`
			}`json:"hour"`
		}`json:"forecastday"`
	}`json:"forecast"`

}

var emojis = map[string]emoji.Emoji{
	"Clear":emoji.Sun,
	"Sunny":emoji.Sun,
	"Patchy rain":emoji.CloudWithRain,
	"Partly cloudy":emoji.SunBehindCloud,
	"Cloudy":emoji.Cloud,
	"Patchy rain nearby":emoji.CloudWithRain,
	"Rainy":emoji.CloudWithRain,
}

func IsSimpleCondition(category string) bool {
    switch category {
    case
        "Sunny",
		"Patchy rain",
        "Partly cloudy",
        "Cloudy",
        "Rainy",
		"Patchy rain nearby",
		"Clear":
        return true
    }
    return false
}

func main() {
 city := "Algiers"
 if len(os.Args) >=2 {
	city = os.Args[1]
 }
 res, err := http.Get("https://api.weatherapi.com/v1/forecast.json?key=94474d04349f43008d395834240102&q="+city+"&days=1&aqi=no&alerts=no")

 if err != nil {
  panic(err)
 }

 defer res.Body.Close()

 if res.StatusCode != 200 {
  panic("Sorry, couldn't get weather for"+city)
 }

 body, err := io.ReadAll(res.Body)

 if err!= nil {
  panic(err)
 }
 var weather Weather
 err = json.Unmarshal(body,&weather)
 if err != nil {
	panic(err)
 }
 location, current, hours := weather.Location, weather.Current, weather.Forecast.Forecastday[0].Hour
 var emj emoji.Emoji
 currentCondition := strings.Trim(current.Condition.Text, " ") 
 if IsSimpleCondition(currentCondition) {
	emj = emojis[currentCondition]
 }else {
	emj = emoji.Cloud
 }
 fmt.Printf(
	"%s, %s: %.0f C, %s %s\n",
    location.Name,location.Country,
	current.TempC,
	currentCondition,
	emj,
 )

 for _,hour := range hours {
	date := time.Unix(hour.TimeEpoch,0)
	// skip hours that are before the current hour
	if date.Before(time.Now()){
		continue
	}
	var emjhr emoji.Emoji
	condition := strings.Trim(hour.Condition.Text, " ") 
	if IsSimpleCondition(condition ) {
		emjhr = emojis[condition]
	}else {
		emjhr = emoji.Cloud
	}
	message := fmt.Sprintf(
		"%s - %.0f C°, %.0f%%, %s %s\n",
		date.Local().Format("15:04"),
		hour.TempC,
		hour.ChanceOfRain,
		condition,
		emjhr,
	)
	
	if(hour.ChanceOfRain >= 50){
		color.Red(message)
	}else{
		fmt.Print(message)
	}
 }
}