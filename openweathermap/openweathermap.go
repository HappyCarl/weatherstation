package openweathermap

import (
	"io/ioutil"
	"fmt"
	"net/http"
	"net/url"
	"bytes"
	"strings"
	base64 "encoding/base64"
)

func Base64Encode(input string) (string) {
	return base64.StdEncoding.EncodeToString([]byte(input))
}


func Transmit(temperature string, humidity string, wind_speed string, rain_1h string, rain_24h string, station_name string, username string, password string) {
	data := url.Values{}

	if(temperature != "") {
		data.Set("temp", temperature)
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

	url := "http://openweathermap.org/data/post"

	auth_data := Base64Encode(strings.Join([]string{username, ":", password}, ""))

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data.Encode()))
	req.Header.Set("Authorization", strings.Join([]string{"Basic", auth_data}, ""))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

}
