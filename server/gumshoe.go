package main

import (
	"bufio"
	"expvar"
	"flag"
	"fmt"
	"log"
	//"net/url"
	"os"
	"path/filepath"
	//"strings"
	"time"

	"github.com/ev1lm0nk3y/gumshoe/config"
	//"github.com/ev1lm0nk3y/gumshoe/misc"
	"github.com/ev1lm0nk3y/gumshoe/server/db"
	"github.com/ev1lm0nk3y/gumshoe/server/fetcher"
	"github.com/ev1lm0nk3y/gumshoe/server/irc"
	"github.com/ev1lm0nk3y/gumshoe/server/matcher"
)

var (
	// Program defaults that are only used when no other data is provided. Based on docker paths.
	defaultGumshoeBase = "/gumshoe"
	defaultCfg         = "cfg/gumshoe.cfg"
	defaultPort        = "9119"
	logger             *log.Logger

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
	fe *fetcher.Client
	ic *irc.Client
	ma *matcher.Matcher
	lg *log.Logger

	Update chan bool
	State  chan config.ProcessState
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

func setupLogging(tc *config.TrackerConfig) *log.Logger {
	tcd := tc.Directories
	logfile := filepath.Join(tcd["user_dir"], tcd["log_dir"], "gumshoe.log")
	if os.Getenv("USER") == "root" {
		logfile = "/var/log/gumshoe.log"
	}

	l, err := os.OpenFile(logfile, 0666, os.O_CREATE|os.O_APPEND|os.O_RDWR)
	if err != nil {
		log.Println("Log file creation failed, using STDOUT: %v", err)
		return log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	}

	w := bufio.NewWriter(l)
	newLog := log.New(w, "", log.Ldate|log.Ltime|log.Lshortfile)
	newLog.Println("Log file started.")
	return newLog
}

func initialize(tc *config.TrackerConfig) (*Gumshoe, error) {
	var err error
	l := setupLogging(tc)
	f := fetcher.New(tc, l)
	m := matcher.New(tc, l, f.GetURL)
	i := irc.New(tc, l, m.CheckMessage)

	tcd := tc.Directories
	dbPath := filepath.Join(tcd["user_dir"], tcd["data_dir"], "gumshoe.db")
	if os.Getenv("USER") == "root" {
		dbPath = "/usr/local/gumshoe/data/gumshoe.db"
	}
	if err = db.InitDb(dbPath); err != nil {
		return nil, fmt.Errorf("[FAIL] Database init failed: %s\n", err)
	}

	g := &Gumshoe{
		fe: f,
		ma: m,
		lg: l,
		ic: i,
		tc: tc,
	}
	return g, nil
}

func (g *Gumshoe) run() error {
	// still need to listen on a port to report stats
	starttime.Set(time.Now().Unix())
	g.lg.Println("Starting Gumshoe.")
	controlChan := make(chan string, 10)
	if err := g.ic.Run(); err != nil {
		g.lg.Fatalf("Unable to start IRC watcher: %v", err)
		return fmt.Errorf("Failed at IRC start")
	}
	for {
		select {
		case s := <-g.Update:
			g.lg.Println("Update to config files detected.")
			g.lock.Lock()
			g.tc = copy(s)
			g.lock.Unlock()
			g.fe.Update()
			g.ma.Update()
			g.ic.Update()
		}
	}
}

func main() {
	var err error
	flag.Parse()

	tc := config.NewTrackerConfig()

	defCfg := filepath.Join(*userConfigDir, "gumshoe.cfg")
	if os.Getenv("USER") == "root" {
		defCfg = "/usr/local/gumshoe/gumshoe.conf"
	}

	if tc, err = LoadUserOrDefaultConfig(defCfg); err != nil {
		log.Fatalln(err)
	}

	defCfgFD, err := os.Open(defCfg, os.O_RDWR)
	if err != nil {
		log.Fatal(err)
	}
	defer tc.Write(defCfgFD)
	defer defCfgFD.Close()

	if *gumshoeDir != tc.GetDirectory("gumshoe_dir") {
		tc.SetGumshoeDirectory("gumshoe_dir", *gumshoeDir)
	}
	if *userConfigDir != tc.GetDirectory("user_dir") {
		tc.SetGumshoeDirectory("user_dir", *userConfigDir)
	}

	gs, err := initialize(tc)
	if err != nil {
		log.Fatalln(err)
	}
	defer gs.lg.Println("Gumshoe exited")

	err = gs.run()
	if err != nil {
		gs.lg.Fatal(err)
	}
}
