package main

import (
  "github.com/tarm/goserial"
  "code.google.com/p/gcfg"
  "log"
  "flag"
  "bytes"
  "strings"
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

func Upload(data string, cfg Config) {
  split := strings.Split(data, ";")

  temperature_1 := strings.Replace(split[3], ",", ".", 1)
  temperature_2 := strings.Replace(split[19], ",", ".", 1)
  humidity_1    := split[11]
  humidity_2    := split[20]
  wind_speed    := split[21]
  rain_ticks    := split[22]
  rain          := split[23]

  // TODO: Calculate average values
  // TODO: Calculate Rain Values
  owm.Transmit(temperature_1, humidity_1, wind_speed, "0", "0", cfg.OpenWeatherMap.StationName, cfg.OpenWeatherMap.Username, cfg.OpenWeatherMap.Password)

  log.Print("Temp 1: " + string(temperature_1) +" Temp 2: " + string(temperature_2) +" Humidity 1: " + humidity_1 +" Humidity 2: " + humidity_2 +" WindSpeed: " + wind_speed +" Rain Ticks: " + rain_ticks + " Rain: " + rain)
}

func ParseConfig(configFile string) (error, Config) {
  var cfg Config

  err := gcfg.ReadFileInto(&cfg, configFile)

  return err, cfg
}
