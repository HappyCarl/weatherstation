package openweathermap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"bytes"
)


func Transmit(temperature string, humidity string, wind_speed string, rain_1h string, rain_24h string) {
	var buffer bytes.Buffer

	buffer.WriteString("http://openweathermap.org/data/post?")

	if(temperature != "") {
		buffer.WriteString("temp=")
		buffer.WriteString(temperature)
		buffer.WriteString("&")
	}

	if(wind_speed != "") {
		buffer.WriteString("wind_speed=")
		buffer.WriteString(wind_speed)
		buffer.WriteString("&")
	}

	if(rain_1h != "") {
		buffer.WriteString("rain_1h=")
		buffer.WriteString(rain_1h)
		buffer.WriteString("&")
	}

	if(rain_24h != "") {
		buffer.WriteString("rain_24h=")
		buffer.WriteString(rain_24h)
		buffer.WriteString("&")
	}

	url := buffer.String()

	fmt.Println(url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer())
	req.Header.Set("Authorization", "Basic *DATA HERE*")

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
