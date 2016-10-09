package main

import (
	"bufio"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/ev1lm0nk3y/gumshoe/db"
	"github.com/ev1lm0nk3y/gumshoe/fetcher"
	_ "github.com/ev1lm0nk3y/gumshoe/http"
	"github.com/ev1lm0nk3y/gumshoe/irc"
	"github.com/ev1lm0nk3y/gumshoe/matcher"
	"github.com/ev1lm0nk3y/gumshoe/misc"
)

var (
	// Program defaults that are only used when no other data is provided. Based on docker paths.
	defaultGumshoeBase = "/gumshoe"
	defaultCfg         = "cfg/gumshoe.cfg"
	defaultPort        = "9119"

	concurrentFetches = make(chan int, 10)
	tc_updated        = make(chan bool)

	port          = flag.String("p", defaultPort, "Which port do we serve requests from. 0 allows the system to decide.")
	gumshoeDir    = flag.String("d", defaultGumshoeBase, "Base path for gumshoe.")
	userConfigDir = flag.String("c", filepath.Join(os.Getenv("HOME"), ".gumshoe"), "User config directory")
	debug         = flag.Bool("debug", false, "Show more logging messages.")

	starttime = expvar.NewInt("started")
)

type Gumshoe struct {
	tc *config.TrackerConfig
	ic *irc.IrcClient
	ma *matcher.Matcher
	ff *fetcher.FileFetch
	lg *log.Logger
	// HTTPService *http.HttpService
}

// Things to do
// * read config
// * start db
// * start logger
// * start the IRC watcher
// - start irc controller that
//   * takes messages from the irc msgs and sends it to the matcher.
//   * matcher calls db to determine if episode is something to fetch.
//   * informs the IRC watcher when the config has been updated.
// - start matcher controller that
//   - listens for channel updates
//   - verifies that this line is something we care about
//   - tells fetcher to get the needed file

func (g *Gumshoe) setupLogging() error {
	tcd := g.tc.Directories
	l, err := os.Create(filepath.Join(tcd["user_dir"], tcd["log_dir"], "gumshoe.log"))
	if err != nil {
		misc.PrintDebugln("Unable to open log file. Will just use stdout")
		return err
	}
	defer l.Close()

	w := bufio.NewWriter(l)
	g.lg = log.New(w, "", log.Ldate|log.Ltime|log.Lshortfile)
	g.lg.Println("Gumshoe logfile started at %s", time.Now().String())
	return nil
}

//TODO(ev1lm0nk3y): re-enable this when the service is running smoothly via
//that command line.  func HttpWatcher(hw *http.HttpControlChannel) {
//	for {
//		select {
//		case update := <-hw.UpdatedCfg:
//			if update {
//				tempTC := *tc
//				err := LoadUserOrDefaultConfig(filepath.Join(tc.Directories["user_dir"], "gumshoe.cfg"))
//				if err != nil {
//					log.Printf("[Error] Unable to successfully load config file. %s\n", err)
//					tc = &tempTC
//					return
//				}
//				tc_updated <- true
//			}
//		case _ = <-hw.NumConnected:
//			// TODO(ev1lm0nk3y): What to do with this number?
//			continue
//		}
//	}
//}

func (g *Gumshoe) Director() {
	for {
		select {
		case msg := <-g.ic.IrcClient.Message:
			if msg != nil {
				continue
			}
			g.ma.AnnChan <- msg
		case tcu := <-tc_updated:
			newTC := LoadUserOrDefaultConfig(filepath.Join(*userConfigDir, "gumshoe.cfg"))
			newG, err := GumshoeInit(newTC)
			if err != nil {
				g.lg.Printf("[WARNING] %v - %v", time.Now().String(), err)
				g.lg.Printf("[WARNING] %v - Reverting to old version", time.Now().String())
				continue
			}
			g = newG
		case iErr := <-g.ic.IRCError:
			if iErr != nil {
				g.lg.Printf("[ERROR] %s - %v\n", time.Now().String, iErr)
			}
		case oc := <-g.ma.OutChan:
			switch oc {
			case matcher.Error:
				g.lg.Printf("[ERROR] %s - Errors occured during validation: %v", time.Now().String(), oc)
			case matcher.Accept:
				link := <-g.ma.Link
				fName := strings.Split(link.Path, "/")
				tcd := g.tc.Directories
				go func(link *url.URL) {
					if err := g.getIt(link, filepath.Join(tcd["user_dir"], tcd["fetch_dir"])); err != nil {
						g.lg.Printf("[ERROR] %s - Errors occured during fetching %s: %v", time.Now().String(), link.String(), err)
					}
				}(link)
			}
		}
	}
}

func (g *Gumshoe) startWatchers() error {
	var err error
	for k, v := range g.tc.Operations.WatchMethods {
		if v {
			switch k {
			case "irc":
				g.ic, err = irc.Init(&tc.IRC, g.lg)
				if err != nil {
					return err
				}
				if g.tc.IRC.EnableLog {
					g.ic.EnableFullIrcLogs()
				}
				if err = g.ic.Start(); err != nil {
					return err
				}
				go g.IrcWatcher()
				return nil
			}
		}
	}
	return fmt.Errorf("No Watchers configured.")
}

//func StartHttp(ic irc.Logger) *http.HttpControlChannel {
//	httpController := http.StartHTTPServer(tc, ic) // Add the logger here too
//	go HttpWatcher(httpController)
//	return httpController
//}

//func KillSignalWatcher(i *irc.IrcClient) {
//	c := make(chan os.Signal, 2)
//	signal.Notify(c, os.Interrupt, os.Kill)
//
//	sig := <-c
//	log.Printf("Signal Received %s. Shutting down gumshoe.\n", sig)
//	i.Enabled <- false
//	h.HttpRunning <- false
//}

// GumshoeInit returns *Gumshoe.
func GumshoeInit(tc *config.TrackerConfig) (*Gumshoe, error) {
	var err error
	var g *Gumshoe
	if tc.Operations.EnableLog {
		if err = g.setupLogging(); err != nil {
			return nil, fmt.Errorf("[ERROR] Writing to the log file failed: %v\n", err)
		}
	}
	g.tc = tc
	g.ic = irc.Init(tc.IRC, g.Logs)
	g.ma = matcher.New(tc.IRC.AnnounceRegexp, tc.IRC.EpisodeRegexp, g.Logs)

	tcd := tc.Directories
	if err = db.InitDb(filepath.Join(tcd["user_dir"], tcd["data_dir"], "gumshoe.db")); err != nil {
		return nil, fmt.Errorf("[FAIL] Database init failed: %s\n", err)
	}
	return g, nil
}

func main() {
	starttime.Set(time.Now().Unix())
	flag.Parse()

	if *debug {
		misc.SetDebug()
	}

	tc = config.NewTrackerConfig()

	tc, err := LoadUserOrDefaultConfig(filepath.Join(*userConfigDir, "gumshoe.cfg"))
	if err != nil {
		log.Fatalln(err)
	}

	//if *port != tc.Operations.HttpPort {
	//	tc.SetGumshoePort(*port)
	//}
	if *gumshoeDir != tc.GetDirectory("gumshoe_dir") {
		tc.SetGumshoeDirectory("gumshoe_dir", *gumshoeDir)
	}
	if *userConfigDir != tc.GetDirectory("user_dir") {
		tc.SetGumshoeDirectory("user_dir", *userConfigDir)
	}
	gs, err := GumshoeInit(tc)
	if err != nil {
		log.Fatalln(err)
	}

	//iW, iLog, err := StartWatcher()
	if err = g.startWatchers(); err != nil {
		log.Fatal(err)
	}

	//hW := StartHttp(iLog)
	//KillSignalWatcher(iW, hW)
}
