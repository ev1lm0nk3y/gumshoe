package gumshoe

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	tc      *TrackerConfig
	cj      []*http.Cookie
	cfgFile string
)

// The primary structure holding the config data, which is read from the preferrences file.
// Options can be changed from the web app or directly in the file. A file watcher will update
// the configuration automatically when the prferrence file is modified.
type IRCChannel struct {
	ChannelOwner   string `json:"owner"`
	Nick           string `json:"nick"`
	Registered     bool   `json:"registered"`
	Key            string `json:"key"`
	Server         string `json:"server"`
	InviteCmd      string `json:"invite_cmd"`
	WatchChannel   string `json:"watch_channel"`
	KeepAlive      int    `json:"keep_alive"`
	PingFreq       int    `json:"ping_frequency"`
	Port           int    `json:"port"`
	Timeout        int    `json:"timeout"`
	EnableLog      bool   `json:"log_irc"`
	AnnounceRegexp string `json:"announce_regex"`
	EpisodeRegexp  string `json:"episode_regex"`
}

type RSSFeed struct {
	URL           string `json:"url"`
	HttpMethod    string `json:http_method"`
	Passkey       string `json:"passkey"`
	Uid           string `json:"uid"`
	RssTtl        int    `json:"ttl"`
	UseServerTtl  bool   `json:"use_server_ttl"`
	EpisodeRegexp string `json:"episode_regex"`
	CookieFile    string `json:"cookie_file"`
}

type Operations struct {
	Email        string          `json:"email"`
	EnableLog    bool            `json:"enable_logging"`
	Debug        bool            `json:"log_debug"`
	EnableWeb    bool            `json:"enable_web"`
	HttpPort     string          `json:"http_port"`
	WatchMethods map[string]bool `json:"watch_methods"`
}

type Download struct {
	Tracker    string `json:"tracker"`
	Rate       int    `json:"download_rate"`
	MaxRetries int    `json:"max_retries"`
	QueueSize  int    `json:"queue_size"`
	Secure     bool   `json:"is_secure"`
}

type TrackerConfig struct {
	Directories  map[string]string `json:"dir_options"`
	Download     Download          `json:"download_params"`
	IRC          IRCChannel        `json:"irc_channel"`
	LastModified int64             `json:"last_modified"`
	Operations   Operations        `json:"operations"`
	// RSS          RSSChannel        `json:"rss_channel"`
}

func NewTrackerConfig() *TrackerConfig {
	return &TrackerConfig{}
}

func (tcfg *TrackerConfig) SetGlobalTrackerConfig() {
	tc = tcfg
}

func (tc *TrackerConfig) String() string {
	output, _ := json.MarshalIndent(&tc, "", "\t")
	return string(output)
}

func (tc *TrackerConfig) CreateDefaultConfig() {
	tc.Directories = map[string]string{
		"base_dir":    os.Getenv("HOME"),
		"data_dir":    "data",
		"log_dir":     "log",
		"torrent_dir": "files",
	}
	tc.Operations = Operations{
		EnableLog: false,
		EnableWeb: true,
		HttpPort:  "8080",
		WatchMethods: map[string]bool{
			"irc": false,
			"rss": false,
		},
	}
	tc.LastModified = time.Now().Unix()
}

func (tc *TrackerConfig) LoadGumshoeConfig(c string) error {
	if err := tc.ProcessGumshoeCfgFile(c); err != nil {
		tc.CreateDefaultConfig()
		return fmt.Errorf("Error with config file %s: %s\nUsing empty template", c, err)
	}
	cfgFile = c
	return nil
}

func (tc *TrackerConfig) ProcessGumshoeCfgFile(c string) error {
	if cfgBuf, err := ioutil.ReadFile(c); err != nil {
		return fmt.Errorf("Error reading file: %s", err)
	} else {
		if err = json.Unmarshal(cfgBuf, &tc); err != nil {
			return fmt.Errorf("Error unmarshaling configs: %s", err)
		}
	}
	if err := tc.SetTrackerCookies(); err != nil {
		return fmt.Errorf("Error setting cookiejar: %s", err)
	}
	return nil
}

func (tc *TrackerConfig) ProcessGumshoeCfgJson(j []byte) error {
	err := json.Unmarshal(j, &tc)
	if err != nil {
		return fmt.Errorf("Invalid JSON: %s", err)
	}
	err = tc.SetTrackerCookies()
	if err != nil {
		return fmt.Errorf("Error setting cookiejar: %s", err)
	}
	return nil
}

func (tc *TrackerConfig) WriteGumshoeConfig(update []byte, f string) error {
	// This is for tests. The normal config file name is as follows.
	cFile := "gumshoe_config.json"
	if f != "" {
		cFile = f
	}
	fullCfgFilepath := filepath.Join(tc.Directories["user_dir"], cFile)

	err := json.Unmarshal(update, &tc)
	if err != nil {
		return err
	}
	ioutil.WriteFile(fullCfgFilepath, []byte(tc.String()), 0655)
	return nil
}

type tempCookies struct {
	Cookies []map[string]string
}

// TODO(ryan): Learn a bit more about encryption, these files shouldn't just
// be lying around.
func (tc *TrackerConfig) SetTrackerCookies() error {
	if !tc.Download.Secure {
		return nil
	}
	// decrypt file here
	cjBuf, err := ioutil.ReadFile(CreateLocalPath("tracker.cj"))
	if err != nil {
		return err
	}

	cookies := &tempCookies{}
	err = json.Unmarshal(cjBuf, &cookies)
	if err != nil {
		return err
	}

	for _, cookie := range cookies.Cookies {
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
		cj = append(cj, c)
	}
	return nil
}

func GetTrackerCookies() []*http.Cookie {
	return cj
}

// An easy utility to generate the fully qualified path name of a given filename
// in the user's data directory.
func CreateLocalPath(fn string) string {
	return filepath.Join(tc.Directories["user_dir"], tc.Directories["data_dir"], fn)
}
