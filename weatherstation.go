package main

import (
  "github.com/tarm/goserial"
  "code.google.com/p/gcfg"
  "log"
  "flag"
  "bytes"
  "strings"
  "strconv"
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

  temperature_1, _ := strconv.ParseFloat(strings.Replace(split[3], ",", ".", 1), 32)
  temperature_2, _ := strconv.ParseFloat(strings.Replace(split[19], ",", ".", 1), 32)
  temperature      := strconv.FormatFloat((temperature_1 + temperature_2) / 2, 'g', 1, 64)

  humidity_1, _    := strconv.ParseFloat(split[11], 32)
  humidity_2, _    := strconv.ParseFloat(split[20], 32)
  humidity         := strconv.FormatFloat((humidity_1 + humidity_2) / 2, 'g', 1, 64)

  wind_speed       := split[21]

  rain_ticks       := split[22]
  rain             := split[23]

  // TODO: Calculate Rain Values
  owm.Transmit(temperature, humidity, wind_speed, "0", "0", cfg.OpenWeatherMap.StationName, cfg.OpenWeatherMap.Username, cfg.OpenWeatherMap.Password)

  log.Print("Temp: " + string(temperature) +" Humidity: " + humidity +" WindSpeed: " + wind_speed +" Rain Ticks: " + rain_ticks + " Rain: " + rain)
}

func ParseConfig(configFile string) (error, Config) {
  var cfg Config

  err := gcfg.ReadFileInto(&cfg, configFile)

  return err, cfg
}
