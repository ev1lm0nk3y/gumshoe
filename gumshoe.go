package gumshoe

import (
	"flag"
	"log"
	"net/http"
	"os"
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
	tc     = config_parser.TrackerConfig{}
	home   = os.Getenv("HOME")
	user   = os.Getenv("USER")
	gopath = os.Getenv("GOPATH")
	gcstat = debug.GCStats{}
)

func GumshoeHandlers() http.Handler {
	gumshoe_handlers := http.NewServeMux()
	gumshoe_handlers.Handle("/", http.FileServer(http.Dir(*base_dir)))
	return gumshoe_handlers
}

type GumshoeSignals struct {
	config_modified chan bool
	shutdown        chan bool
	logger          chan Logger
	tc              chan TrackerConfig
	patterns        chan Patterns
}

func init() {
	flag.Parse()
	if err := tc.LoadGumshoeConfig(*config_file); err != nil {
		log.Fatal(err)
	}
	if tc.Operations.HttpPort != *port && tc.Operations.HttpPort != "" {
		if err := flag.Set("p", tc.Operations.HttpPort); err != nil {
			log.Println(err)
		}
	}
	signals := make(GumshoeSignals)
}

func main() {
	go startHttpServer()
	go irc.StartIRCClient(&tc)
}

func startHttpServer() {
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
