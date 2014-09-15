package gumshoe

import (
	"flag"
	"log"
	"net/http"
	"os"
  "path"
	"path/filepath"
	"runtime/debug"
	"time"
)

// HTTP Server Flags
var port = flag.String("p", "http",
	"Which port do we serve requests from. 0 allows the system to decide.")
var base_dir = flag.String("base_dir",
	"/home/ryan/Development/bitme-get/src/gumshoe/http",
	"Base path for the HTTP server's files.")

// Base Config Stuff
var config_file = flag.String("c",
	filepath.Join(os.Getenv("HOME"), ".gumshoe", "config.json"),
	"Location of the configuration file.")

// Get this flag set working!
var (
	tc     = TrackerConfig{}
	home   = os.Getenv("HOME")
	user   = os.Getenv("USER")
	gopath = os.Getenv("GOPATH")
  gumshoeSrc = os.Getenv("GUMSHOESRC")
	gcstat = debug.GCStats{}
)

func GumshoeHandlers() http.Handler {
	gumshoe_handlers := http.NewServeMux()
	gumshoe_handlers.Handle("/", http.FileServer(path.Join(gumshoeSrc, "html")))
	return gumshoe_handlers
}

type GumshoeSignals struct {
	config_modified chan bool
	shutdown        chan bool
	// logger          chan Logger
	tcSignal        chan TrackerConfig
	showSignal      chan Shows
}

func init() {
	flag.Parse()
  if gumshoeSrc == "" {
    gumshoeSrc = "/home/ryan/gocode/src/gumshoe"
  }
	if err := tc.LoadGumshoeConfig(*config_file); err != nil {
		log.Fatal(err)
	}
	if tc.Operations.HttpPort != *port && tc.Operations.HttpPort != "" {
		if err := flag.Set("p", tc.Operations.HttpPort); err != nil {
			log.Println(err)
		}
	}
	signals := new(GumshoeSignals)

  allShows := NewShowsConfig()
  if numShows, err := allShows.LoadShows(); err == nil {
    log.Sprintf("You have %d shows that you are tracking.", numShows)
  }

}

func main() {
  // go StartMetrics()
	go StartHttpServer()
	go StartIRC()
}

func StartHttpServer() {
	s := &http.Server{
		Addr:           "127.0.0.1:" + *port,
		Handler:        GumshoeHandlers(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Println("Starting up webserver...")
	log.Fatal(s.ListenAndServe())
}
