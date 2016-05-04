package main

import (
  "bufio"
  "expvar"
  "log"
  "net/http"
  "os"
  "path/filepath"
  "regexp"
  "strconv"

	"github.com/coopernurse/gorp"
  "github.com/thoj/go-ircevent"
)


var (
  // Program defaults that are only used when no other data is provided.
  DEFAULT_GUMSHOE_BASE = "/usr/local/gumshoe"
  DEFAULT_CFG = "/usr/local/gumshoe/default.cfg"
  DEFAULT_PORT = "20123"

	concurrentFetches = make(chan int, 10)
	fetchResultMap = expvar.NewMap("fetch_results").Init() // map of fetch return code counters
	lastFetch      = expvar.NewInt("last_fetch_timestamp") // timestamp of last successful fetch

	tc  *TrackerConfig
  tc_updated = make(chan bool)  // Those systems that can be dynamically updated, should watch this channel.
	cj  []*http.Cookie
	gDb *gorp.DbMap
  cfgFile string
  httpPort string

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
  tc = NewTrackerConfig()
  lastFetch.Set(int64(0))
}

func SetUserConfigFile(c string) {
  cfgFile = c
}

func SetGumshoeBaseDirectory(d string) {
  tc.Directories["gumshoe_dir"] = d
}

func SetGumshoePort(p int) {
  httpPort = strconv.Itoa(p)
}

func loadConfig() error {
  userCfg := filepath.Join(os.Getenv("HOME"), ".gumshoe", "gumshoe.cfg")
  if cfgFile == "" {
    // Try to find the user's config first.
    cfgFile = userCfg
    _, err := os.Stat(cfgFile)
    if err != nil {
      cfgFile = DEFAULT_CFG
    }
  }
  err := tc.LoadGumshoeConfig(cfgFile)
  if err != nil {
    tc.Directories["gumshoe_dir"] = DEFAULT_GUMSHOE_BASE
    tc.Directories["user_dir"] = userCfg
    tc.Operations.HttpPort = httpPort
  }
  return err
}

func setupLogging() (logger *log.Logger, err error) {
  l, err := os.Create(filepath.Join(tc.Directories["user_dir"], tc.Directories["log_dir"], "gumshoe.log"))
  if err != nil {
    PrintDebugln("Unable to open log file. Will just use stdout")
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
      PrintDebugln("Updating gumshoe configuration.")
      // Put update function calls below here
      updateEpisodeRegex()
      // Put update function calls above here
      tc_updated<- false
    }
  }
}

func Start() (err error) {
  err = loadConfig()
  if err != nil {
    log.Fatalf("[FAIL] Unable to load config: %s\n", err)
  }

  go UpdateAllComponents()
  tc_updated<- true

  // Unified logging is nice, but not necessary right now.
  //if tc.Operations.EnableLog {
  //  logger, err := setupLogging()
  //}
  //if err != nil {
  //  log.Printf("[ERROR] Writing to the log file failed: %s", err)
  //}

  err = InitDb()
  if err != nil {
    log.Fatalf("[FAIL] Database init failed: %s\n", err)
  }

  for k, v := range tc.Operations.WatchMethods {
    if v {
      switch k {
      case "irc":
        log.Println("Starting IRC Watcher.")
        StartIRC()  // Add the logger here
      default:
        PrintDebugf("%s is coming soon.\n", k)
      }
    }
  }

  StartHTTPServer(tc.Directories["gumshoe_dir"], tc.Operations.HttpPort)  // Add the logger here too
  log.Println("Exiting Gumshoe.")
  return err
}
