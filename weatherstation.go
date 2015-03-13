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
  owm "./openweathermap"
  )

type Config struct {
  Communication struct {
    Port string
  }
  OpenWeatherMap struct {
    StationName string
    Username string
    Password string
  }
}

var last_update = time.Now()
//Every 1 min a value is saved
var rain_1h_array [60]int
var rain_1h_index int = 0

var rain_24h_array [60*24]int
var rain_24h_index int = 0


func main() {
  var configFile = flag.String("config", "config.gcfg", "Path to Config-File.")

  flag.Parse()


  log.Print("Reading Config...")
  err, cfg := ParseConfig(*configFile)

  if err != nil {
    log.Fatal(err)
    return
  }

  log.Print("Config has been read.")


  StartCommunication(cfg.Communication.Port, cfg)
}

func StartCommunication(port string, cfg Config) {
  c := &serial.Config{Name: port, Baud: 9600}

  log.Print("Starting Communication")


  s, err := serial.OpenPort(c)
  if err != nil || s == nil {
    log.Fatal(err)
    return
  }

  log.Print("Port has been opened.")


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

    Upload(data, cfg)
  }
}

func Convert(data string) (float64, error) {
  return strconv.ParseFloat(strings.TrimSpace(strings.Replace(strings.Replace(data, ",", ".", 1), "\x00", "", 42)), 64)
}

func Upload(data string, cfg Config) {
  log.Print("Received Data: ", data)
  split := strings.Split(data, ";")

  temperature_1, err1 := Convert(split[3])
  temperature_2, err2 := Convert(split[19])
  temperature         := (temperature_1 + temperature_2) / 2

  if err1 != nil && err2 == nil {
    temperature = temperature_1
  } else if err1 == nil && err2 != nil {
    temperature = temperature_2
  }

  temperature_s := strconv.FormatFloat(temperature, 'g', 1, 64)

  humidity_1, errA    := Convert(split[11])
  humidity_2, errB    := Convert(split[20])
  humidity            := (humidity_1 + humidity_2) / 2

  if errA != nil && errB == nil {
    humidity = humidity_1
  } else if errA == nil && errB != nil {
    humidity = humidity_2
  }

  humidity_s := strconv.FormatFloat(humidity, 'g', 1, 64)

  wind_speed       := split[21]

  //TODO: Calc Rain 4 real
  //∆1minute, put into db (or local, ggf arraylist)
  // WS/(2*A(in cm^2, 86.6))

  rain_ticks, _ := Convert(split[22])

  rain_1h, rain24h := calculateRain(int(rain_ticks))
  rain             := split[23]

  // TODO: Calculate Rain Values
  owm.Transmit(temperature_s, humidity_s, wind_speed, rain_1h, rain_24h, cfg.OpenWeatherMap.StationName, cfg.OpenWeatherMap.Username, cfg.OpenWeatherMap.Password)

  log.Print("Temp: " + temperature_s +" Humidity: " + humidity_s +" WindSpeed: " + wind_speed +" Rain Ticks: " + rain_ticks + " Rain 1h: " + rain_1h + " Rain 24h: " + rain_24h_ + " Rain: " + rain)
}

/*
Calculates the fallen rain

1 rain tick means that 5ml water fell. the ticks are absolute and reset after reaching 4096 to 0
The area collecting the rain is 86.6cm² big
The amount of rain in mm is calculated with (rain_ticks/2*(86.6cm²)) (blame Tony Metger if this does not work)

when a new rain_tick count is received, it
  - calculates the time passed since last update
  - calculates delta ticks with rain_1h[index]
  - calculates ticks/minute with delta ticks and passed time
  - overwrites rain_1h/24h[index+1] until rain_1h/24h[index+1+int(passed time)] with tick/minute
  - adds all values in the array to get tick count/ time frame(1h/24h)
  - uses math described above
  - prays to Linux Torvalds for making Linux possible (press Alt+F4 to pray now)

returns rain_1h and rain_24h
 */
func calculateRain(rain_ticks int) (float64, float64) {

  //time since last run
  minutes_since_last_run := int(time.Since(last_update).Minutes())
  last_update = time.Now()

  //delta ticks
  var rain_ticks_delta int = 0
  if(rain_1h_array[rain_1h_index] > rain_ticks) {
    rain_ticks_delta = ((4096 + rain_ticks) - rain_1h_array[rain_1h_index])
  } else {
    rain_ticks_delta = (rain_ticks - rain_1h_array[rain_1h_index])
  }

  //ticks/minute
  rain_ticks_minute := int(rain_ticks_delta / minutes_since_last_run)

  //update values in arrays
  for i := rain_1h_index; i <= rain_1h_index + minutes_since_last_run; i++ {
    rain_1h_array[i % len(rain_1h_array)] = rain_ticks_minute
  }
  rain_1h_index = (rain_1h_index + minutes_since_last_run) % len(rain_1h_array)

  for i := rain_24h_index; i <= rain_24h_index + minutes_since_last_run; i++ {
    rain_24h_array[i % len(rain_1h_array)] = rain_ticks_minute
  }
  rain_24h_index = (rain_24h_index + minutes_since_last_run) % len(rain_1h_array)

  //sum the arrays
  rain_1h_sum := 0
  for i := 0; i < len(rain_1h_array); i++ {
    rain_1h_sum += rain_1h_array[i];
  }

  rain_24h_sum := 0
  for i := 0; i < len(rain_24h_array); i++ {
    rain_24h_sum += rain_24h_array[i];
  }
  //calculate fallen rain
  rain_1h := rain_1h_sum / (2*86.6)
  rain_24h := rain_24h_sum / (2*86.6)

  return rain_1h,rain_24h
}

func ParseConfig(configFile string) (error, Config) {
  var cfg Config

  err := gcfg.ReadFileInto(&cfg, configFile)

  return err, cfg
}
