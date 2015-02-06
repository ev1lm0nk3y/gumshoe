package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var defaultConfigFile = filepath.Join(os.Getenv("HOME"), ".gumshoe", "config.json")
var configFile = flag.String("c", defaultConfigFile,
	"Location of the configuration file.")

var usageString = `Usage: gumshoe-cli [options] <op> 

op:
  add <url>     - add url to download queue
  test <string> - test string against patterns, add to download queue if match found
  status        - server status
  patterns      - show configured patterns
  config        - show config

options:
  -c            - location of the configuration file
                  [default: %s]
`

func usage() {
	fmt.Printf(usageString, defaultConfigFile)
	os.Exit(2)
}

func main() {
	flag.Parse()

	// TODO(deekue) replace println with actual gumshoe function calls
	switch flag.Arg(0) {
	case "add":
		println("add url to download queue", flag.Arg(1))
		os.Exit(0)
	case "test":
		println("test string", flag.Arg(1))
		os.Exit(0)
	case "status":
		println("status")
		os.Exit(0)
	case "p", "pat", "patterns":
		println("pattern show")
		os.Exit(0)
	case "config":
		println("config show")
		os.Exit(0)
	}
	usage()
}
