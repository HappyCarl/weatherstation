package main

import (
	"bytes"
	"code.google.com/p/gcfg"
	"flag"
	"fmt"
	"github.com/tarm/goserial"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	//owm "./openweathermap"
)

//Config is the data type for the config file
type Config struct {
	Data struct {
		Latitude  string
		Longitude string
	}
	Communication struct {
		Port     string
		BaudRate int
	}
	OpenWeatherMap struct {
		StationName string
		Username    string
		Password    string
	}
	Webserver struct {
		Address string
	}
}

var lastUpdate = time.Now()

//Every 1 min a value is saved
var rain1hArray [60]int
var rain1hIndex int

var rain24hArray [60 * 24]int
var rain24hIndex int

var firstData = true

//current values, ment to be served by the http server
var currentTemp float64
var currentHumidity float64
var currentSpeed float64
var currentRain bool
var currentRain1h float64
var currentRain24h float64

func main() {
	var configFile = flag.String("config", "config.gcfg", "Path to Config-File.")

	flag.Parse()

	//Read and parse the cofig file
	log.Print("Reading Config...")
	cfg, err := ParseConfig(*configFile)

	if err != nil {
		log.Fatal(err)
		return
	}

	log.Print("Config has been read.")

	//start the serial communication
	go StartCommunication(cfg)

	//start the web server
	StartWebserver(cfg)
}

//StartWebserver starts and initializes the web server to serve the weather data
func StartWebserver(cfg Config) {
	log.Print("Starting webserver on " + cfg.Webserver.Address)
	http.HandleFunc("/data", DataHTTPHandler)
	http.HandleFunc("/debug", DebugHTTPHandler)
	http.Handle("/", http.FileServer(assetFS()))
	http.ListenAndServe(cfg.Webserver.Address, nil)
}

//DataHTTPHandler handles the rain data requests
func DataHTTPHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	log.Print("HTTP: data requested")
	fmt.Fprintf(w, "{\"temp\": %.1f,\"humidity\": %.0f,\"wind_speed\": %.1f,\"rain\": {\"h1\": %.1f, \"h24\": %.1f, \"current\": %t }}", currentTemp, currentHumidity, currentSpeed, currentRain1h, currentRain24h, currentRain)
}

//DebugHTTPHandler returns some debug statistics
func DebugHTTPHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("HTTP: debug requested")
	fmt.Fprintf(w, "Rain1h: %s \nRain24h: %s", ArrayToString(rain1hArray[:]), ArrayToString(rain24hArray[:]))
}

//StartCommunication opens the serial connection and reads the data from it
func StartCommunication(cfg Config) {
	//initialize the serial connection
	c := &serial.Config{Name: cfg.Communication.Port, Baud: cfg.Communication.BaudRate}

	log.Print("Starting Communication")

	//and now open the port
	s, err := serial.OpenPort(c)
	if err != nil || s == nil {
		log.Fatal(err)
		return
	}

	log.Print("Port has been opened.")

	//endlosschleife
	for true {
		var buffer bytes.Buffer
		stop := false

		for !stop {
			buf := make([]byte, 4)
			_, err = s.Read(buf)
			if err != nil {
				log.Fatal(err)
			} else {
				value := string(buf)
				buffer.WriteString(value)
				stop = value[0] == '\015'
			}
		}

		data := buffer.String()
		data = strings.TrimSpace(data)

		//parse the incoming data
		Parse(data, cfg)
	}
}

//Convert transformes the string data to the corresponding float64 value
func Convert(data string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(strings.Replace(strings.Replace(data, ",", ".", 1), "\x00", "", 42)), 64)
}

//Parse thates the received data and parses it
func Parse(data string, cfg Config) {
	log.Print("Received Data: ", data)
	split := strings.Split(data, ";")

	//calculate average temperature
	temperature1, err1 := Convert(split[3])
	temperature2, err2 := Convert(split[19])
	temperature := (temperature1 + temperature2) / 2

	if err1 != nil && err2 == nil {
		temperature = temperature1
	} else if err1 == nil && err2 != nil {
		temperature = temperature2
	}

	temperatureString := strconv.FormatFloat(temperature, 'f', 16, 64)

	//calculate average humdity
	humidity1, errA := Convert(split[11])
	humidity2, errB := Convert(split[20])
	humidity := (humidity1 + humidity2) / 2

	if errA != nil && errB == nil {
		humidity = humidity1
	} else if errA == nil && errB != nil {
		humidity = humidity2
	}

	humidityString := strconv.FormatFloat(humidity, 'f', 16, 64)

	//get the wind speed
	p, _ := Convert(split[21])
	windSpeed := strconv.FormatFloat(p, 'f', 16, 64)

	//and calculate the rain
	rainTicks, _ := Convert(split[22])
	rainTicksString := strconv.FormatFloat(rainTicks, 'f', 16, 64)

	rain1h, rain24h := calculateRain(int(rainTicks))
	rain1hString := strconv.FormatFloat(rain1h, 'f', 16, 64)
	rain24hString := strconv.FormatFloat(rain24h, 'f', 16, 64)
	rain := split[23]

	//owm upload currently not working
	//owm.Transmit(temperatureString, humidityString, windSpeed, rain1hString, rain24hString, cfg.OpenWeatherMap.StationName, cfg.OpenWeatherMap.Username, cfg.OpenWeatherMap.Password, cfg.Data.Longitude, cfg.Data.Latitude)

	//save all the data to serve them via HTTP
	currentTemp = temperature
	currentHumidity = humidity
	currentSpeed = p
	currentRain = (rain == "1")
	currentRain1h = rain1h
	currentRain24h = rain24h

	log.Print("Temp: " + temperatureString + " Humidity: " + humidityString + " WindSpeed: " + windSpeed + " Rain Ticks: " + rainTicksString + " Rain 1h: " + rain1hString + " Rain 24h: " + rain24hString + " Rain: " + rain)
}

/*
Calculates the fallen rain

1 rain tick means that 5ml water fell. the ticks are absolute and reset after reaching 4096 to 0
The area collecting the rain is 86.6cm² big
So at first we calculate the the ticks in the last 1/24h and multiply it by 0.005, having the amount of L water that fell on the funnel
We then divide that by 0.00866m^2 and multiply it by 1m^2 to get the amount of water that fell on a square meter
1L on a square meter equals 1mm high water

So the formula is '(ticks * 0.005)/0.00866'

when a new rain_tick count is received, it
  - calculates the time passed since last update
  - writes the rain ticks into the array for the passed time
  - calculates the fallen rain ticks
  - uses math described above
  - prays to Linus Torvalds for making Linux possible (press Alt+F4 to pray now)

returns rain1h and rain24h
*/
func calculateRain(rainTicks int) (float64, float64) {

	if firstData {
		firstData = false

		for i := 0; i < len(rain1hArray); i++ {
			rain1hArray[i] = rainTicks
		}

		for i := 0; i < len(rain24hArray); i++ {
			rain24hArray[i] = rainTicks
		}
	}

	//time since last run
	minutesSinceLastRun := int(time.Since(lastUpdate).Minutes())
	//horrible hack for first run, when receiving data without passing a minute
	if minutesSinceLastRun == 0 {
		minutesSinceLastRun = 1
	}

	lastUpdate = time.Now()

	//update values in arrays
	for i := rain1hIndex; i <= rain1hIndex+minutesSinceLastRun; i++ {
		rain1hArray[i%len(rain1hArray)] = rainTicks
	}
	rain1hIndex = (rain1hIndex + minutesSinceLastRun) % len(rain1hArray)

	for i := rain24hIndex; i <= rain24hIndex+minutesSinceLastRun; i++ {
		rain24hArray[i%len(rain1hArray)] = rainTicks
	}
	rain24hIndex = (rain24hIndex + minutesSinceLastRun) % len(rain1hArray)

	//now takes the current rain tick and the 1/24 hour ago value and calculates the delta value, representing the fallen rain
	rain1hDelta := rain1hArray[rain1hIndex] - rain1hArray[(rain1hIndex+1)%len(rain1hArray)]

	rain24hDelta := rain24hArray[rain24hIndex] - rain24hArray[(rain24hIndex+1)%len(rain24hArray)]

	//calculate fallen rain
	//rain1h := float64(rain1hDelta) / (2 * 86.6) old and wrong formula
	rain1h := ((float64(rain1hDelta) * 0.005) / 0.00866)
	//rain24h := float64(rain24hDelta) / (2 * 86.6)
	rain24h := ((float64(rain24hDelta) * 0.005) / 0.00866)

	return rain1h, rain24h
}

//ParseConfig does exactly what the name says...
func ParseConfig(configFile string) (Config, error) {
	var cfg Config

	err := gcfg.ReadFileInto(&cfg, configFile)

	return cfg, err
}

func ArrayToString(data []int) string {

	result := "["
	for _, value := range data {
		result += strconv.FormatInt(int64(value), 10) + ","
	}
	result += "]"
	return result
}
