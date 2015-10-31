package main

import (
	"expvar"
	"flag"
	"os"
	"runtime/debug"
  "strconv"
	"strings"

	"github.com/ev1lm0nk3y/gumshoe/gumshoe"
)

// HTTP Server Flags
var port = flag.String("p", "20123",
	"Which port do we serve requests from. 0 allows the system to decide.")
var baseDir = flag.String("d", "/usr/local/gumshoe", "Base path for gumshoe.")
var quiet = flag.Bool("q", false,
	"Supress log messages.")

// Base Config Stuff
var configFile = flag.String("c", "",
	"Location of the configuration file. Default is $HOME/.gumshoe/gumshoe.cfg")

// TODO Get this flag set working!
var (
	tc         = gumshoe.NewTrackerConfig()
	home       = os.Getenv("HOME")
	user       = os.Getenv("USER")
	gopath     = os.Getenv("GOPATH")
	gumshoeSrc = os.Getenv("GUMSHOESRC")
	gcstat     = debug.GCStats{}
)

// Metrics
var (
	argv                = expvar.NewString("argv")
	watchLastUpdateTime = expvar.NewInt("watch_updated_timestamp")
	httpPort            = expvar.NewString("port")
)

func init() {
	argv.Set(strings.Join(os.Args, " "))
	watchLastUpdateTime.Set(int64(0))
}

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
