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
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
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
	Database struct {
		Connection string
	}
}

type WeatherData struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Temperature float64
	Humidity float64
	Windspeed float64
	Raining bool
	RainTicks int
}
var dbSql *sql.DB
var db gorm.DB

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

	//Setting up the database
	dbSql, err = sql.Open("mysql", cfg.Database.Connection)
	if err != nil {
		//Abort
		log.Fatal(err)
		return
	}
	db , err = gorm.Open("mysql", dbSql)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	db.CreateTable(&WeatherData{})

	//start the serial communication
	go StartCommunication(cfg)

	//start the web server
	StartWebserver(cfg)
}



//StartWebserver starts and initializes the web server to serve the weather data
func StartWebserver(cfg Config) {
	log.Print("Starting webserver on " + cfg.Webserver.Address)
	http.HandleFunc("/data", DataHTTPHandler)
	//http.Handle("/", http.FileServer(assetFS()))
	http.ListenAndServe(cfg.Webserver.Address, nil)
}

//DataHTTPHandler handles the rain data requests
func DataHTTPHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	log.Print("HTTP: data requested")

	//get the latest record
	var record = WeatherData{}
  res := db.Order("created_at desc").First(&record)
	if res.Error != nil {
		fmt.Fprint(w, "{\"error\": \"Something went wrong when querying the database!\"}")
		log.Print("Something went wrong:")
		log.Print(res)
		return
	}
	now := time.Now()

	rain1h := CalculateRain(GetRainTicksSince(now, now.Add(-1 * time.Hour)))
	rain24h := CalculateRain(GetRainTicksSince(now, now.Add(-24 * time.Hour)))

	fmt.Fprintf(w, "{\"temp\": %.1f,\"humidity\": %.0f,\"wind_speed\": %.1f,\"rain\": {\"h1\": %.1f, \"h24\": %.1f, \"current\": %t }}", record.Temperature, record.Humidity, record.Windspeed, rain1h, rain24h, record.Raining)
}

func GetRainTicksSince(t1, t2 time.Time) (int){
	log.Print(t2)
	weatherData := []WeatherData{}
	res := db.Where("created_at BETWEEN ? AND ?", t2, t1).Order("created_at desc").Find(&weatherData)
	if res.Error != nil {
		log.Print("An error occured getting the weatherdata")
		return -1
	}
	var overflowCounter = 0
	var lastTicks = weatherData[0].RainTicks
	var initTicks = weatherData[len(weatherData) -1].RainTicks
	for _, dataset := range weatherData {
		if dataset.RainTicks > lastTicks {
			log.Print("OVERFLOW now at " + strconv.Itoa(overflowCounter) + " (" + strconv.Itoa(dataset.RainTicks) + ">" + strconv.Itoa(lastTicks) + ")")
			overflowCounter++
		}
		lastTicks = dataset.RainTicks
	}
	log.Print("Found " + strconv.Itoa(overflowCounter) + " overflows")
	var result = overflowCounter * 4096 + weatherData[0].RainTicks - initTicks
	log.Printf("%d Ticks calculated (endTicks=%d, startTicks=%d)", result, weatherData[0].RainTicks, initTicks)
	return result
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

//Parse takes the received data and parses it
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

	//calculate average humdity
	humidity1, errA := Convert(split[11])
	humidity2, errB := Convert(split[20])
	humidity := (humidity1 + humidity2) / 2

	if errA != nil && errB == nil {
		humidity = humidity1
	} else if errA == nil && errB != nil {
		humidity = humidity2
	}

	//get the wind speed
	windSpeed, _ := Convert(split[21])

	//and get the rain
	rainTicks, _ := Convert(split[22])

	//rain1h, rain24h := calculateRain(int(rainTicks))
	rain := split[23]

	//Put all the data into the database
	//Sometimes there are erroros, where everything is 0
	if humidity != 0 && int(rainTicks) != 0 {
		weatherdata := WeatherData{Temperature: temperature, Humidity: humidity, Windspeed: windSpeed, Raining: (rain == "1"), RainTicks: int(rainTicks)}
		db.Create(&weatherdata)
	}


	log.Printf("Temp: %.1f Humidity: %.0f WindSpeed: %.1f RainTicks: %.0f Rain: %b", temperature, humidity, windSpeed, rainTicks, currentRain)
}

/*
Calculates the fallen rain

1 rain tick means that 5ml water fell. the ticks are absolute and reset after reaching 4096 to 0
The area collecting the rain is 86.6cm² big
So we get the ticks and multiply it by 0.005, having the amount of L water that fell on the funnel
We then divide that by 0.00866m^2 to get the amount of water that fell on a square meter(Unit L/m^2 = mm)
1L on a square meter equals 1mm high water

So the formula is '(ticks * 0.005)/0.00866'

returns rain
*/
func CalculateRain(rainTicks int) (float64) {
	//calculate fallen rain
	//rain1h := float64(rain1hDelta) / (2 * 86.6) old and wrong formula
	rain := ((float64(rainTicks) * 0.005) / 0.00866)

	return rain
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
