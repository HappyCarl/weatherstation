package openweathermap

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"log"
	base64 "encoding/base64"
)

func Base64Encode(input string) (string) {
	return base64.StdEncoding.EncodeToString([]byte(input))
}


func Transmit(temperature string, humidity string, wind_speed string, rain_1h string, rain_24h string, station_name string, username string, password string, longitude string, latitude string) {
	data := url.Values{}

	if(temperature != "") {
		data.Set("temp", temperature)
	}
	
	if(humidity != "") {
		data.Set("humidity", humidity)	
	}

	if(wind_speed != "") {
		data.Set("wind_speed", wind_speed)
	}

	if(rain_1h != "") {
		data.Set("rain_1h", rain_1h)
	}

	if(rain_24h != "") {
		data.Set("rain_24h", rain_24h)
	}
	
	if(station_name != "") {
		data.Set("name", station_name)
	}

	data.Set("lat", latitude)
	data.Set("long", longitude)
	data.Set("alt", "17")

	url := "http://openweathermap.org/data/post"

	auth_data := Base64Encode(strings.Join([]string{username, ":", password}, ""))
	log.Print("Sending: ", data.Encode())
	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	req.Header.Set("Authorization", strings.Join([]string{"Basic", auth_data}, " "))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if(resp.StatusCode == 200) {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Print("Successfully uploaded data. ", resp.Status, " Body: ", string(body))
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Print("Failed to transmit Data. ", resp.Status, " Body: ", string(body))
	}

}
