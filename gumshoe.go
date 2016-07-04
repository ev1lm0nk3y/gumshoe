package main

import (
	"bufio"
	"errors"
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/ev1lm0nk3y/gumshoe/db"
	"github.com/ev1lm0nk3y/gumshoe/fetcher"
	"github.com/ev1lm0nk3y/gumshoe/http"
	"github.com/ev1lm0nk3y/gumshoe/irc"
	"github.com/ev1lm0nk3y/gumshoe/misc"
)

var (
	// Program defaults that are only used when no other data is provided. Based on docker paths.
	DEFAULT_GUMSHOE_BASE = "/gumshoe"
	DEFAULT_CFG          = "cfg/gumshoe.cfg"
	DEFAULT_PORT         = "9119"

	concurrentFetches = make(chan int, 10)
	tc                *config.TrackerConfig
	tc_updated        = make(chan bool)

	port          = flag.String("p", DEFAULT_PORT, "Which port do we serve requests from. 0 allows the system to decide.")
	gumshoeDir    = flag.String("d", DEFAULT_GUMSHOE_BASE, "Base path for gumshoe.")
	userConfigDir = flag.String("c", filepath.Join(os.Getenv("HOME"), ".gumshoe"), "User config directory")
	debug         = flag.Bool("debug", false, "Show more logging messages.")

	starttime = expvar.NewInt("started")
)

func init() {
	tc = config.NewTrackerConfig()
	starttime.Set(time.Now().Unix())
}

func LoadUserOrDefaultConfig(c string) error {
	err := tc.LoadGumshoeConfig(c)
	if err == nil {
		return nil
	}
	log.Println(err)
	log.Printf("Error loading config %s. Trying the default.\n", c)
	err = tc.LoadGumshoeConfig(filepath.Join(DEFAULT_GUMSHOE_BASE, DEFAULT_CFG))
	if err != nil {
		log.Println("Default config is invalid.")
	}
	return err
}

func setupLogging() (logger *log.Logger, err error) {
	l, err := os.Create(filepath.Join(tc.Directories["user_dir"], tc.Directories["log_dir"], "gumshoe.log"))
	if err != nil {
		misc.PrintDebugln("Unable to open log file. Will just use stdout")
		return
	}
	defer l.Close()

	w := bufio.NewWriter(l)
	logger = log.New(w, "", log.Ldate|log.Ltime|log.Lshortfile)
	log.Println("Gumshoe logfile started.")
	return
}

func UpdateAllComponents() {
	misc.PrintDebugln("Updating gumshoe configuration.")
	db.SetEpisodePatternRegexp(tc.IRC.EpisodeRegexp)
}

func Start() error {
	// Unified logging is nice, but not necessary right now.
	//if tc.Operations.EnableLog {
	//  logger, err := setupLogging()
	//}
	//if err != nil {
	//  log.Printf("[ERROR] Writing to the log file failed: %s", err)
	//}
	err := db.InitDb(filepath.Join(tc.Directories["user_dir"], tc.Directories["data_dir"], "gumshoe.db"))
	if err != nil {
		return errors.New(fmt.Sprintf("[FAIL] Database init failed: %s\n", err))
	}
	db.SetEpisodePatternRegexp(tc.IRC.EpisodeRegexp)
	db.SetEpisodeQualityRegexp("720|1080")

	for k, v := range tc.Operations.WatchMethods {
		if v {
			switch k {
			case "irc":
				ic, err := irc.InitIrc(&tc.IRC)
				if err != nil {
					return err
				}
				if err = ic.StartIRC(); err != nil {
					return err
				}
				go IrcWatcher(ic.IrcControlChannel)
			default:
				misc.PrintDebugf("%s is coming soon.\n", k)
			}
		}
	}

	log.Printf("Gumshoe http starting on port %s", tc.Operations.HttpPort)
	http.StartHTTPServer(tc.Directories["gumshoe_dir"], tc.Operations.HttpPort, tc) // Add the logger here too
	return err
}

func main() {
	flag.Parse()

	if *debug {
		misc.SetDebug()
	}
	err := LoadUserOrDefaultConfig(filepath.Join(*userConfigDir, "gumshoe.cfg"))
	if err != nil {
		log.Fatalln(err)
	}

	if *port != tc.Operations.HttpPort {
		tc.SetGumshoePort(*port)
	}
	if *gumshoeDir != tc.GetDirectory("gumshoe_dir") {
		tc.SetGumshoeDirectory("gumshoe_dir", *gumshoeDir)
	}
	if *userConfigDir != tc.GetDirectory("user_dir") {
		tc.SetGumshoeDirectory("user_dir", *userConfigDir)
	}
	err = Start()
	if err != nil {
		log.Fatal(err)
	}
}

func IrcWatcher(control *irc.IrcControlChannel) {
	for {
		select {
		case match := <-control.IRCAnnounceMatch:
			if match != nil {
				ep, err := db.CheckMatch(match[1])
				if err != nil {
					log.Println(err)
					continue
				}
				ff, err := fetcher.NewFileFetch(match[2], filepath.Join(tc.Directories["user_dir"], tc.Directories["torrent_dir"]), tc.CookieJar)
				if err != nil {
					log.Println(err)
					continue
				}
				err = ff.RetrieveEpisode()
				if err != nil {
					log.Printf("FAIL: episode not retrieved: %s\n", err)
					continue
				}
				err = ep.AddEpisode()
				if err != nil {
					log.Printf("Episode is already downloading: %s\n", err)
				}
			}
		case tcu := <-tc_updated:
			control.IRCConfigChanged <- tcu
		case irc_err := <-control.IRCError:
			if irc_err != nil {
				log.Println(irc_err)
			}
		}
	}
}
