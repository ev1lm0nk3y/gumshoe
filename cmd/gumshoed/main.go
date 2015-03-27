package main

import (
  "expvar"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
  "strings"
  "time"

	"github.com/ev1lm0nk3y/gumshoe/gumshoe"
)

// HTTP Server Flags
var port = flag.String("p", "http",
	"Which port do we serve requests from. 0 allows the system to decide.")
var baseDir = flag.String("d",
	filepath.Join(os.Getenv("HOME"), ".local", "gumshoe"),
	"Base path for gumshoe.")
var quiet = flag.Bool("q", false,
  "Supress log messages.")

// Base Config Stuff
var configFile = flag.String("c", "",
	"Location of the configuration file. Default is $HOME/.local/gumshoe/config.json")

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
  argv = expvar.NewString("argv")
  watchLastUpdateTime = expvar.NewInt("watch_updated_timestamp")
  httpPort = expvar.NewString("port")
)

func init() {
  argv.Set(strings.Join(os.Args, " "))
  watchLastUpdateTime.Set(int64(0))
}

func main() {
	flag.Parse()
  log.SetFlags(log.LstdFlags | log.Lshortfile)
  if *configFile == "" {
    flag.Set("c", filepath.Join(*baseDir, "config.json"))
  }
  log.Printf("Reading config %s", *configFile)
	if err := tc.LoadGumshoeConfig(*configFile); err != nil {
		log.Fatal(err)
	}
	if tc.Operations.HttpPort != *port && tc.Operations.HttpPort != "" {
		if err := flag.Set("p", tc.Operations.HttpPort); err != nil {
			log.Fatal(err)
		}
	}
  httpPort.Set(tc.Operations.HttpPort)

	log.Println("Starting up gumshoe...")
	gumshoe.InitShowDb(*baseDir)

	//DEBUG
	gumshoe.LoadTestData()

	// start enabled watchers
	for k, v := range tc.Operations.WatchMethods {
		if v {
			switch k {
			case "rss":
				log.Println("pretending to starting RSS watcher")
			case "irc":
				log.Println("starting IRC watcher")
        watcher, err := gumshoe.StartIRC(tc)
        if err != nil {
          log.Println(err)
          log.Println("IRC config unusable. Update config and try again.")
        }
        log.Printf("IRC connection to %s is %s", watcher.Server, watcher.Connected())
			case "log":
				log.Println("pretending to starting log file watcher")
			}
      watchLastUpdateTime.Set(time.Now().Unix())
		}
	}

	gumshoe.StartHTTPServer(*baseDir, *port)
	log.Println("Exiting gumshoe...")
}
