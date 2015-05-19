package main

import (
  "github.com/tarm/goserial"
  "code.google.com/p/gcfg"
  "log"
  "flag"
  "bytes"
  "strings"
  "strconv"
  "time"
  "net/http"
  "fmt"
  //owm "./openweathermap"
  )

//data type for the config file
type Config struct {
  Data struct {
    Latitude string
    Longitude string
  }
  Communication struct {
    Port string
    BaudRate int
  }
  OpenWeatherMap struct {
    StationName string
    Username string
    Password string
  }
  Webserver struct {
    Address string
  }
}

var last_update = time.Now()
//Every 1 min a value is saved
var rain_1h_array [60]int
var rain_1h_index int = 0

var rain_24h_array [60*24]int
var rain_24h_index int = 0

var first_data bool = true

//current values, ment to be served by the http server
var current_temp float64
var current_humidity float64
var current_speed float64
var current_rain bool
var current_rain_1h float64
var current_rain_24h float64


func main() {
  var configFile = flag.String("config", "config.gcfg", "Path to Config-File.")

  flag.Parse()

  //Read and parse the cofig file
  log.Print("Reading Config...")
  err, cfg := ParseConfig(*configFile)

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

func StartWebserver(cfg Config) {
  log.Print("Starting webserver on " + cfg.Webserver.Address)
  http.HandleFunc("/data", DataHttpHandler)
  http.Handle("/", http.FileServer(assetFS()))
  http.ListenAndServe(cfg.Webserver.Address, nil)
}

func DataHttpHandler(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Access-Control-Allow-Origin", "*")
  log.Print("HTTP: data requested")
  fmt.Fprintf(w,"{\"temp\": %.1f,\"humidity\": %.0f,\"wind_speed\": %.1f,\"rain\": {\"h1\": %.1f, \"h24\": %.1f, \"current\": %t }}", current_temp, current_humidity, current_speed, current_rain_1h, current_rain_24h, current_rain)
}


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

func Convert(data string) (float64, error) {
  return strconv.ParseFloat(strings.TrimSpace(strings.Replace(strings.Replace(data, ",", ".", 1), "\x00", "", 42)), 64)
}

func Parse(data string, cfg Config) {
  log.Print("Received Data: ", data)
  split := strings.Split(data, ";")

  //calculate average temperature
  temperature_1, err1 := Convert(split[3])
  temperature_2, err2 := Convert(split[19])
  temperature         := (temperature_1 + temperature_2) / 2

  if err1 != nil && err2 == nil {
    temperature = temperature_1
  } else if err1 == nil && err2 != nil {
    temperature = temperature_2
  }

  temperature_s := strconv.FormatFloat(temperature, 'f', 1, 64)

  //calculate average humdity
  humidity_1, errA    := Convert(split[11])
  humidity_2, errB    := Convert(split[20])
  humidity            := (humidity_1 + humidity_2) / 2

  if errA != nil && errB == nil {
    humidity = humidity_1
  } else if errA == nil && errB != nil {
    humidity = humidity_2
  }

  humidity_s := strconv.FormatFloat(humidity, 'f', 1, 64)

  //get the wind speed
  p, _ := Convert(split[21])
  wind_speed       := strconv.FormatFloat(p, 'f', 1, 64)


  //and calculate the rain
  rain_ticks, _ := Convert(split[22])
  rain_ticks_s := strconv.FormatFloat(rain_ticks, 'f', 1, 64)

  rain_1h, rain_24h := calculateRain(int(rain_ticks))
  rain_1h_s := strconv.FormatFloat(rain_1h, 'f', 1, 64)
  rain_24h_s := strconv.FormatFloat(rain_24h, 'f', 1, 64)
  rain             := split[23]

  //owm upload currently not working
  //owm.Transmit(temperature_s, humidity_s, wind_speed, rain_1h_s, rain_24h_s, cfg.OpenWeatherMap.StationName, cfg.OpenWeatherMap.Username, cfg.OpenWeatherMap.Password, cfg.Data.Longitude, cfg.Data.Latitude)

  //save all the data to serve them via HTTP
  current_temp = temperature
  current_humidity = humidity
  current_speed = p
  current_rain = (rain == "1")
  current_rain_1h = rain_1h
  current_rain_24h = rain_24h

  log.Print("Temp: " + temperature_s +" Humidity: " + humidity_s +" WindSpeed: " + wind_speed +" Rain Ticks: " + rain_ticks_s + " Rain 1h: " + rain_1h_s + " Rain 24h: " + rain_24h_s + " Rain: " + rain)
}

/*
Calculates the fallen rain

1 rain tick means that 5ml water fell. the ticks are absolute and reset after reaching 4096 to 0
The area collecting the rain is 86.6cm² big
The amount of rain in mm is calculated with (rain_ticks/2*(86.6cm²)) (blame Tony Metger if this does not work)

when a new rain_tick count is received, it
  - calculates the time passed since last update
  - writes the rain ticks into the array for the passed time
  - calculates the fallen rain ticks
  - uses math described above
  - prays to Linus Torvalds for making Linux possible (press Alt+F4 to pray now)

returns rain_1h and rain_24h
 */
func calculateRain(rain_ticks int) (float64, float64) {

  if first_data {
    first_data = false
	
	for i := 0; i < len(rain_1h_array); i++ {
	  rain_1h_array[i] = rain_ticks
	}
	
	for i := 0; i < len(rain_24h_array); i++ {
	  rain_24h_array[i] = rain_ticks
	}
  }

  //time since last run
  minutes_since_last_run := int(time.Since(last_update).Minutes())
  //horrible hack for first run, when receiving data without passing a minute
  if(minutes_since_last_run == 0) {
    minutes_since_last_run = 1
  }

  last_update = time.Now()


  //update values in arrays
  for i := rain_1h_index; i <= rain_1h_index + minutes_since_last_run; i++ {
    rain_1h_array[i % len(rain_1h_array)] = rain_ticks
  }
  rain_1h_index = (rain_1h_index + minutes_since_last_run) % len(rain_1h_array)

  for i := rain_24h_index; i <= rain_24h_index + minutes_since_last_run; i++ {
    rain_24h_array[i % len(rain_1h_array)] = rain_ticks
  }
  rain_24h_index = (rain_24h_index + minutes_since_last_run) % len(rain_1h_array)

  
  //now takes the current rain tick and the 1/24 hour ago value and calculates the delta value, representing the fallen rain 
  rain_1h_delta := rain_1h_array[rain_1h_index] - rain_1h_array[(rain_1h_index + 1) % len(rain_1h_array)]
  

  rain_24h_delta := rain_24h_array[rain_24h_index] - rain_24h_array[(rain_24h_index + 1) % len(rain_24h_array)]

  //calculate fallen rain
  rain_1h := float64(rain_1h_delta) / (2*86.6)
  rain_24h := float64(rain_24h_delta) / (2*86.6)

  return rain_1h,rain_24h
}

func ParseConfig(configFile string) (error, Config) {
  var cfg Config

  err := gcfg.ReadFileInto(&cfg, configFile)

  return err, cfg
}
