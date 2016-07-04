package main

import (
	"bufio"
	_ "errors"
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	errChan           = make(chan string, 5)

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

func HttpWatcher(hw *http.HttpControlChannel) {
	for {
		select {
		case update := <-hw.UpdatedCfg:
			if update {
				tempTC := *tc
				err := LoadUserOrDefaultConfig(filepath.Join(tc.Directories["user_dir"], "gumshoe.cfg"))
				if err != nil {
					log.Printf("[Error] Unable to successfully load config file. %s\n", err)
					tc = &tempTC
					return
				}
				tc_updated <- true
			}
		case _ = <-hw.NumConnected:
			// TODO(ev1lm0nk3y): What to do with this number?
			continue
		}
	}
}

func IrcWatcher(ic *irc.IrcControlChannel) {
	for {
		select {
		case match := <-ic.IRCAnnounceMatch:
			if match != nil {
				ep, err := db.CheckMatch(match[1])
				if err != nil {
					errChan <- err.Error()
					continue
				}
				ff, err := fetcher.NewFileFetch(match[2], filepath.Join(tc.Directories["user_dir"], tc.Directories["torrent_dir"]), tc.CookieJar)
				if err != nil {
					errChan <- err.Error()
					continue
				}
				err = ff.RetrieveEpisode()
				if err != nil {
					errChan <- fmt.Sprintf("FAIL: episode not retrieved: %s\n", err.Error())
					continue
				}
				err = ep.AddEpisode()
				if err != nil {
					errChan <- fmt.Sprintf("Episode is already downloading: %s\n", err.Error())
				}
			}
		case tcu := <-tc_updated:
			ic.IRCConfigChanged <- tcu
		case irc_err := <-ic.IRCError:
			if irc_err != nil {
				errChan <- irc_err.Error()
			}
		}
	}
}

func StartWatcher() (*irc.IrcControlChannel, irc.Logger, error) {
	for k, v := range tc.Operations.WatchMethods {
		if v {
			switch k {
			case "irc":
				ic, err := irc.InitIrc(&tc.IRC)
				if err != nil {
					return nil, nil, err
				}
				if err = ic.StartIRC(); err != nil {
					return nil, nil, err
				}
				go IrcWatcher(ic.IrcControlChannel)
				return ic.IrcControlChannel, ic.GetIrcLogs, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("No Watchers configured.")
}

func StartHttp(ic irc.Logger) *http.HttpControlChannel {
	httpController := http.StartHTTPServer(tc, ic) // Add the logger here too
	go HttpWatcher(httpController)
	return httpController
}

func Setup() error {
	// Unified logging is nice, but not necessary right now.
	//if tc.Operations.EnableLog {
	//  logger, err := setupLogging()
	//}
	//if err != nil {
	//  log.Printf("[ERROR] Writing to the log file failed: %s", err)
	//}
	err := db.InitDb(filepath.Join(tc.Directories["user_dir"], tc.Directories["data_dir"], "gumshoe.db"))
	if err != nil {
		return fmt.Errorf("[FAIL] Database init failed: %s\n", err)
	}
	db.SetEpisodePatternRegexp(tc.IRC.EpisodeRegexp)
	db.SetEpisodeQualityRegexp("720|1080")
	return nil
}

func KillSignalWatcher(i *irc.IrcControlChannel, h *http.HttpControlChannel) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill)

	sig := <-c
	log.Printf("Signal Received %s. Shutting down gumshoe.\n", sig)
	i.IRCEnabled <- false
	h.HttpRunning <- false
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
	err = Setup()
	if err != nil {
		log.Fatalf("Issues setting up: %s\n", err)
	}
	iW, iLog, err := StartWatcher()
	if err != nil {
		log.Fatal(err)
	}

	hW := StartHttp(iLog)
	KillSignalWatcher(iW, hW)
}
