package main

import (
  "github.com/tarm/goserial"
  "code.google.com/p/gcfg"
  "log"
  )

type Config struct {
  Communication struct {
    Port string
  }
}

func main() {
  log.Print("Reading Config...")
  err, cfg := ParseConfig()

  if err != nil {
    log.Fatal(err)
    return
  }

  log.Print("Config has been read.")

  StartCommunication(cfg.Communication.Port)
}

func StartCommunication(port string) {
  c := &serial.Config{Name: port, Baud: 9600}

  log.Print("Starting Communication")

  s, err := serial.OpenPort(c)
  if err != nil || s == nil {
    log.Fatal(err)
    return
  }

  log.Print("Port has been opened.")

  /*
  n, err := s.Write([]byte("test"))
  if err != nil {
    log.Fatal(err)
  }

  buf := make([]byte, 128)
  n, err = s.Read(buf)
  if err != nil {
      log.Fatal(err)
  }
  log.Print("%q", buf[:n])*/
}

func ParseConfig() (error, Config) {
  var cfg Config

  err := gcfg.ReadFileInto(&cfg, "config.gcfg")

  return err, cfg
}
