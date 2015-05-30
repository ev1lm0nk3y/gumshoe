package gumshoe

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"
)

var (
  tc  *TrackerConfig
)

func init() {
  tc = NewTrackerConfig()
}

// The primary structure holding the config data, which is read from the preferrences file.
// Options can be changed from the web app or directly in the file. A file watcher will update
// the configuration automatically when the prferrence file is modified.
type IMDBConfig struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	Uid  int    `json:"uid"`
}

type IRCChannel struct {
	Nick         string `json:"nick"`
	Key          string `json:"key"`
	Server       string `json:"server"`
	ChannelOwner string `json:"channel_owner"`
	NeedInvite   bool   `json:"invite_needed"`
	WatchChannel string `json:"watch_channel"`
	KeepAlive    int    `json:"keep_alive"`
	PingFreq     int    `json:"ping_frequency"`
	IRCPort      int    `json:"irc_port"`
	Timeout      int    `json:"timeout"`
	EnableLog    bool   `json:"enable_logging"`
	LogPath      string
	Debug        bool `json:"debug"`
}

type RSSChannel struct {
	FeedURI      string `json:"feed"`
	Passkey      string `json:"passkey"`
	Uid          string `json:"rss_uid"`
	RssTtl       int    `json:"rss_ttl"`
	UseServerTtl bool   `json:"use_server_ttl"`
}

type Operations struct {
	EnableLog    bool            `json:"enable_logging"`
	EnableWeb    bool            `json:"enable_web"`
	HttpPort     string          `json:"http_port"`
	LogLevel     string          `json:"log_level"`
	UseIMDB      bool            `json:"use_imdb_watchlist"`
	WatchMethods map[string]bool `json:"watch_methods"`
}

type Download struct {
	Tracker    string              `json:"tracker"`
	Rate       int                 `json:"download_rate"`
	MaxRetries int                 `json:"max_retries"`
	QueueSize  int                 `json:"queue_size"`
	Secure     bool                `json:"is_secure"`
	Cookies    []map[string]string `json:"cookies"`
}

type TrackerConfig struct {
	Cookiejar    []*http.Cookie
	Files        map[string]string `json:"file_options"`
  IMDB         IMDBConfig        `json:"imdb_cfg"`
	IRC          IRCChannel `json:"irc_channel"`
  Operations   Operations `json:"operations"`
  RSS          RSSChannel        `json:"rss_channel"`
	Download  `json:"download_params"`
	LastModified int64    `json:"last_modified"`
}

func NewTrackerConfig() *TrackerConfig {
	return &TrackerConfig{}
}

func (tc *TrackerConfig) String() string {
  output, _ := json.MarshalIndent(&tc, "", "\t")
  return string(output)
}

func (tc *TrackerConfig) LoadGumshoeConfig(cfgFile string) error {
	if err := tc.ProcessGumshoeJSON(cfgFile); err != nil {
		log.Println("Error with config file ", cfgFile, ": ", err)
		log.Println("Using basic template.")
		return tc.ProcessGumshoeJSON("config/gumshoe_config.json")
	}
	return nil
}

func (tc *TrackerConfig) ProcessGumshoeJSON(cfgJson string) error {
	if cfgBuf, err := ioutil.ReadFile(cfgJson); err != nil {
		return err
	} else {
		if err := json.Unmarshal(cfgBuf, &tc); err != nil {
			return err
		}
	}
	if tc.Download.Secure {
		tc.SetTrackerCookies()
	}
	return nil
}

func (tc *TrackerConfig) SetTrackerCookies() {
	tc.Cookiejar = nil
	for i := range tc.Download.Cookies {
		cookie := tc.Download.Cookies[i]
		c := &http.Cookie{
			Name:   cookie["Name"],
			Value:  cookie["Value"],
			Path:   cookie["Path"],
			Domain: cookie["Domain"],
		}
		exp, err := strconv.Atoi(cookie["Expires"])
		if err == nil {
			c.Expires = time.Unix(int64(exp), 0)
		} else {
			c.Expires = time.Now().AddDate(10, 0, 0)
		}
		tc.Cookiejar = append(tc.Cookiejar, c)
	}
}

// An easy utility to generate the fully qualified path name of a given filename
// and prepending the base directory and a subdirectory, if given.
func (tc *TrackerConfig) CreateLocalPath(f, s string) string {
	return filepath.Join(tc.Files["base_dir"], s, f)
}

/*
func (tc *TrackerConfig) WriteGumshoeConfig(update []byte) error {
	err := json.Unmarshal(update, &tc)
	if err == nil {
		var gCfg []byte
		gCfg, err := json.MarshalIndent(&tc, "", "  ")
		if err == nil {
      if (tc.Files["base_dir"] != "") {
        base := string(tc.Files["base_dir"])
			  return ioutil.WriteFile(base+"gumshoe_config.json",
				  gCfg, 0655)
      } else {
        log.Println(gCfg)
      }
		}
		return err
	}
	return err
}

func (tc *TrackerConfig) JsonGumshowConfig() ([]byte, error) {
  full := ""
  helpfile, err := ioutil.ReadFile(filepath.Join(tc.Files["base_dir"], "www", "help.xml"))
  if err != nil {
    return nil, err
  }


  return json.MarshalIndent(&tc, "", "  ")
}
*/
