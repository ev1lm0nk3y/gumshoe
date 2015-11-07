package main

import (
	"flag"
  "strconv"

	"github.com/ev1lm0nk3y/gumshoe/gumshoe"
)

var (
  // HTTP Server Flags
  port = flag.String("p", "20123", "Which port do we serve requests from. 0 allows the system to decide.")
  baseDir = flag.String("d", "/usr/local/gumshoe", "Base path for gumshoe.")

  // Base Config Stuff
  configFile = flag.String("c", "",	"Location of the configuration file. Default is $HOME/.gumshoe/gumshoe.cfg")
)

func main() {
	flag.Parse()

  if *configFile != "" {
    gumshoe.SetUserConfigFile(*configFile)
  }

  if *port != "20123" {
    tp, _ := strconv.Atoi(*port)
    gumshoe.SetGumshoePort(tp)
  }
  //if *baseDir != "/usr/local/gumshoe" {
  //  gumshoe.SetGumshoeBaseDirectory(*baseDir)
  //}

  gumshoe.Start()
}
