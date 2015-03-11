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

  peter, _ := Convert(split[22])

  rain_ticks       := strconv.FormatFloat((peter * 0.295), 'g', 1, 64)
  rain             := split[23]

  // TODO: Calculate Rain Values
  owm.Transmit(temperature_s, humidity_s, wind_speed, rain_ticks, rain_ticks, cfg.OpenWeatherMap.StationName, cfg.OpenWeatherMap.Username, cfg.OpenWeatherMap.Password)

  log.Print("Temp: " + temperature_s +" Humidity: " + humidity_s +" WindSpeed: " + wind_speed +" Rain Ticks: " + rain_ticks + " Rain: " + rain)
}

func ParseConfig(configFile string) (error, Config) {
  var cfg Config

  err := gcfg.ReadFileInto(&cfg, configFile)

  return err, cfg
}
