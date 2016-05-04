package main

import (
  "bufio"
  "expvar"
  "flag"
  "log"
  "net/http"
  "os"
  "path/filepath"
  "regexp"
  "strconv"

  "github.com/ev1lm0nk3y/gumshoe/config"
  "github.com/ev1lm0nk3y/gumshoe/db"
  "github.com/ev1lm0nk3y/gumshoe/fetcher"
  "github.com/ev1lm0nk3y/gumshoe/http"
  "github.com/ev1lm0nk3y/gumshoe/irc"
  "github.com/ev1lm0nk3y/gumshoe/misc"

	"github.com/nelsam/gorq"
)


var (
  // Program defaults that are only used when no other data is provided.
  DEFAULT_GUMSHOE_BASE = "/usr/local/gumshoe"
  DEFAULT_CFG = "/usr/local/gumshoe/default.cfg"
  DEFAULT_PORT = "20123"

	concurrentFetches = make(chan int, 10)
	tc  *config.TrackerConfig
  tc_updated = make(chan bool)  // Those systems that can be dynamically updated, should watch this channel.
	cj  []*http.Cookie
	gDb *gorq.DbMap
  cfgFile string
  httpPort string

  // HTTP Server Flags
  port = flag.String("p", DEFAULT_PORT, "Which port do we serve requests from. 0 allows the system to decide.")
  baseDir = flag.String("d", "/usr/local/gumshoe", "Base path for gumshoe.")

  // Base Config Stuff
  configFile = flag.String("c", filepath.Join(os.Getenv("HOME"), ".gumshoe", "data", "gumshoe.cfg"),	"Config file to load")

	// Regexp to determine if the announce regexp matches a known episode structure
	episodePattern *regexp.Regexp
	// Quickly determine the quality of the show with this regex
	episodeQualityRegexp = regexp.MustCompile("720p|1080p")
	// Regexp for messages from IRC channel announcing something to do something about
	announceLine *regexp.Regexp
	// Time, in ms, when the connection to the IRC server was established
	ircConnectTimestamp = expvar.NewInt("irc_connect_timestamp")
	// Time, in ms, when the channel was last updated
	ircUpdateTimestamp = expvar.NewInt("irc_last_update_timestamp")
	// String relating to the current state of the IRC watcher
	ircStatus = expvar.NewString("irc_status")
	// IRC client object
	ircClient *irc.Connection
	// channel that gets timestamp updates for ircUpdateTimestamp in order to ensure we write only the most recent timestamp into that exported variable.
	metricUpdate = make(chan int64)
	// channel that locks the DB while we update it to prevent data corruption.
	checkDBLock = make(chan int)
	// Channel that is used to turn on and off the IRC watcher.
	IRCEnabled = make(chan bool)
	// Channel to signify if the IRC config has changed. Changes will restart the IRC watcher.
	IRCConfigChanged = make(chan bool)
	// Channel that collects all IRC errors and will disconnect the IRC watcher if it encounters one.
	IRCConfigError = make(chan error)
)

func init() {
  flag.Parse()
  tc = config.NewTrackerConfig()
  lastFetch.Set(int64(0))
}

func SetGumshoeBaseDirectory(d string) {
  tc.Directories["gumshoe_dir"] = d
}

func SetGumshoePort(p int) {
  tc.Operations.HttpPort = strconv.Itoa(p)
}

func LoadUserOrDefaultConfig(c string) error {
  err := tc.LoadGumshoeConfig(c)
  if err == nil {
    return
  }
  log.Println(err)
  log.Printf("Error loading config %s. Trying the default.\n", c)
  err = tc.LoadGumshoeConfig(DEFAULT_CFG)
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
  logger = log.New(w, "", log.Ldate | log.Ltime | log.Lshortfile)
  log.Println("Gumshoe logfile started.")
  return
}

func UpdateAllComponents() {
  for {
    tcu := <-tc_updated
    if tcu {
      misc.PrintDebugln("Updating gumshoe configuration.")
      // Put update function calls below here
      updateEpisodeRegex()
      // Put update function calls above here
      tc_updated<- false
    }
  }
}

func Start() (err error) {
  go UpdateAllComponents()
  tc_updated<- true

  // Unified logging is nice, but not necessary right now.
  //if tc.Operations.EnableLog {
  //  logger, err := setupLogging()
  //}
  //if err != nil {
  //  log.Printf("[ERROR] Writing to the log file failed: %s", err)
  //}

  err = db.InitDb()
  if err != nil {
    log.Fatalf("[FAIL] Database init failed: %s\n", err)
  }

  for k, v := range tc.Operations.WatchMethods {
    if v {
      switch k {
      case "irc":
        log.Println("Starting IRC Watcher.")
        icc := irc.StartIRC(*tc)  // Add the logger here
        go IrcWatcher(icc)
      default:
        misc.PrintDebugf("%s is coming soon.\n", k)
      }
    }
  }

  log.Printf("Gumshoe http starting on port %s", tc.Operations.HttpPort)
  StartHTTPServer(tc.Directories["gumshoe_dir"], tc.Operations.HttpPort)  // Add the logger here too
  log.Println("Exiting Gumshoe.")
  return err
}

func main() {
  err := LoadUserOrDefaultConfig(*configFile)
  if err != nil {
    log.Fatalln(err)
  }

  if *port != tc.Operations.HttpPort {
    SetGumshoePort(*port)
  }
  if *baseDir != tc.Directories["gumshoe_dir"] {
    SetGumshoeBaseDirectory(*baseDir)
  }

  gumshoe.Start()
}

func IrcWatcher(control *irc.IRCControlChannel) {
  for {
    select {
    case match := <-control.IRCAnnounceMatch:
      if match != nil {
        err := db.CheckMatch(match[1])
        if err != nil {
          log.Println(err)
          continue
        }
        ff, err := fetcher.NewFileFetch(match[2])
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
    case <-tc_updated:
      if tc_updated {
        control.IRCConfigChanged<- true
      }
    case irc_err := <-IRCConfigError:
      if irc_err != nil {
        log.Println(irc_err)
      }
    }
  }
}
