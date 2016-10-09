// Package config
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

// The primary structure holding the config data, which is read from the preferrences file.
// Options can be changed from the web app or directly in the file. A file watcher will update
// the configuration automatically when the prferrence file is modified.

// IRCChannel contains all the information to connect and log irc chat rooms.
type IRCChannel struct {
	ChannelOwner   string `json:"owner,omitempty"`
	Nick           string `json:"nick,omitempty"`
	Registered     bool   `json:"registered,omitempty"`
	Key            string `json:"key,omitempty"`
	Server         string `json:"server,omitempty"`
	InviteCmd      string `json:"invite_cmd,omitempty"`
	WatchChannel   string `json:"watch_channel,omitempty"`
	KeepAlive      int    `json:"keep_alive,omitempty"`
	PingFreq       int    `json:"ping_frequency,omitempty"`
	Port           int    `json:"port,omitempty"`
	Timeout        int    `json:"timeout,omitempty"`
	EnableLog      bool   `json:"log_irc,omitempty"`
	AnnounceRegexp string `json:"announce_regex,omitempty"`
	EpisodeRegexp  string `json:"episode_regex,omitempty"`
}

// Operations is the base information needed to run gumshoe.
type Operations struct {
	Email        string          `json:"email,omitempty"`
	EnableLog    bool            `json:"enable_logging,omitempty"`
	Debug        bool            `json:"log_debug,omitempty"`
	EnableWeb    bool            `json:"enable_web,omitempty"`
	HttpPort     string          `json:"http_port,omitempty"`
	WatchMethods map[string]bool `json:"watch_methods,omitempty"`
}

// Download controls how gumshoe downloads from a tracker and interfaces with
// your downloader.
type Download struct {
	Tracker     string `json:"tracker,omitempty"`
	Rate        int    `json:"download_rate,omitempty"`
	MaxRetries  int    `json:"max_retries,omitempty"`
	QueueSize   int    `json:"queue_size,omitempty"`
	Secure      bool   `json:"is_secure,omitempty"`
	CookieFile  string `json:"cookies_file,omitempty"`
	TorrentURL  string `json:"torrent_url,omitempty"`
	TorrentUser string `json:"torrent_user,omitempty"`
	TorrentPass string `json:"torrent_pass,omitempty"`
}

// Directories lay out the structure of a user's setup.
type Directories struct {
	Main     string `json:"gumshoe_dir,omitempty"`
	User     string `json:"user_dir,omitempty"`
	Data     string `json:"data_dir,omitempty"`
	Download string `json:"download_dir,omitempty"`
	Fetch    string `json:"fetch_dir,omitempty"`
	Log      string `json:"log_dir,omitempty"`
}

// TrackerConfig holds all the configurations for gumshoe.
type TrackerConfig struct {
	cookieJar    []*http.Cookie
	Directories  Directories `json:"dir_options,omitempty"`
	Download     Download    `json:"download_params,omitempty"`
	IRC          IRCChannel  `json:"irc_channel,omitempty"`
	LastModified int64       `json:"last_modified"`
	Operations   Operations  `json:"operations,omitempty"`
}

var (
	defaultConfigPath = filepath.Join(os.Getenv("HOME"), ".gumshoe")
	defaultConfig     = TrackerConfig{
		Directories: Directories{
			Main:     "/usr/local/gumshoe",
			User:     defaultConfigPath,
			Data:     filepath.Join(defaultConfigPath, "data"),
			Download: filepath.Join(defaultConfigPath, "download"),
			Fetch:    filepath.Join(defaultConfigPath, "fetch"),
			Log:      filepath.Join(defaultConfigPath, "logs"),
		},
		Operations: Operations{
			EnableLog: false,
			EnableWeb: false,
			HttpPort:  "8080",
			WatchMethods: map[string]bool{
				"irc": false,
				"rss": false,
			},
		},
		LastModified: time.Now().Unix(),
	}
)

// New will read in the cfg, and parse the json to return a TrackerConfig. Upon
// errors, a default TrackerConfig will be returned, along with the error, so
// that startup has something to work with.
func New(cfg io.Reader) (*TrackerConfig, error) {
	var tc TrackerConfig
	var err error
	b := new(bytes.Buffer)
	if _, err = b.ReadFrom(cfg); err != nil {
		return &defaultConfig, fmt.Errorf("Config Read Error: %v", err)
	}
	if err = json.Unmarshal(b.Bytes(), &tc); err != nil {
		return &defaultConfig, fmt.Errorf("JSON Unmarshall Failed: %v", err)
	}
	return &tc, tc.postProcess()
}

// Write is an extension of the Writer interface and will dump the TrackConfig
// to the given Writer.
func (tc *TrackerConfig) Write(cfg io.Writer) error {
	b, _ := json.MarshalIndent(&tc, "", "\t")
	_, err := cfg.Write(b)
	return err
}

// Update takes a json byte slice and unmarshalls it onto TrackerConfig.
func (tc *TrackerConfig) Update(update []byte) error {
	var newTC *TrackerConfig
	if err := json.Unmarshal(update, &newTC); err != nil {
		return fmt.Errorf("Update Config Failed: %v", err)
	}
	newTC.LastModified = time.Now().Unix()
	return tc.merge(newTC)
}

// String implements the Stringer interface and will pretty print TrackerConfig,
// with indents and everything.
func (tc *TrackerConfig) String() string {
	output, _ := json.MarshalIndent(&tc, "", "\t")
	return string(output)
}

// Json will return a string of TrackerConfig that has been HTML escaped and
// compacted.
func (tc *TrackerConfig) Json() string {
	output, _ := json.Marshal(&tc)
	b := new(bytes.Buffer)
	json.HTMLEscape(b, output)
	json.Compact(b, b.Bytes())
	return b.String()
}

// Cookies will return the http cookies for your tracker.
func (tc *TrackerConfig) Cookies() []*http.Cookie {
	return tc.cookieJar
}

// SetCookies will update the TrackerConfig cookie jar, and write them to disk.
func (tc *TrackerConfig) SetCookies(cookies io.Reader) error {
	var err error
	var cj *os.File
	if tc.cookieJar, err = readCookieJar(cookies); err != nil {
		return fmt.Errorf("[ConfigError] Updating Cookies: %v", err)
	}
	if cj, err = os.OpenFile(
		filepath.Join(tc.Directories.Data, tc.Download.CookieFile),
		os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600); err != nil {
		return fmt.Errorf("[ConfigError] Cookiejar file error: %v", err)
	}
	cj.Truncate(0)
	cj.Seek(0, 0)
	return writeCookieJar(cj, tc.cookieJar)
}

func (tc *TrackerConfig) merge(newTC *TrackerConfig) *TrackerConfig {
	var final *TrackerConfig
	newElem := reflect.TypeOf(newTC).Elem()
	oldElem := reflect.TypeOf(tc).Elem()
	finalElem := reflect.TypeOf(final).Elem()
	for i := 0; i < oldElem.NumField(); i++ {
		if oldElem.Field(i).Type == reflect.TypeOf(int64(0)) || oldElem.Field(i).Type == reflect.TypeOf([]*http.Cookie{}) {
			finalElem.Field(i).Set(oldElem.Field(i))
			continue
		}
		newfElem := reflect.TypeOf(newElem.Field(i)).Elem()
		oldfElem := reflect.TypeOf(oldElem.Field(i)).Elem()
		finalfElem := reflect.TypeOf(finalElem.Field(i)).Elem()
		for y = 0; y < oldfElem.NumField(); i++ {
			if newfElem.Field(y).IsNil() {
				finalfElem.Field(y).Set(oldfElem.Field(y))
				continue
			}
			oldfElem.Field(y).Set(newfElem.Field(y))
		}
	}
	return final
}

func (tc *TrackerConfig) postProcess() error {
	if tc.Download.Secure {
		cj, err := os.OpenFile(
			filepath.Join(tc.Directories.Data, tc.Download.CookieFile),
			os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
		if err != nil {
			return fmt.Errorf("[cookieJarError] Unable to open cookiejar: %v", err)
		}
		var jar []*http.Cookie
		if jar, err = readCookieJar(cj); err != nil {
			return err
		}
		if !reflect.DeepEqual(jar, tc.cookieJar) {
			cj.Truncate(0)
			cj.Seek(0, 0)
			writeCookieJar(cj, jar)
			tc.cookieJar = jar
		}
	}
	if tc.LastModified == 0 {
		tc.LastModified = time.Now().Unix()
	}
	return nil
}

type cookiejar struct {
	Cookies []map[string]string `json:"cookies"`
}

// ReadcookieJar will attempt to read from io.Reader containing netscape-style
// cookies in json form and returns []*http.Cookie.
func readCookieJar(cj io.Reader) ([]*http.Cookie, error) {
	b := new(bytes.Buffer)
	if _, err := b.ReadFrom(cj); err != nil {
		return nil, fmt.Errorf("[ConfigError] Unable to read cookie input: %v", err)
	}

	var cjar cookiejar
	if err := json.Unmarshal(b.Bytes(), &cjar); err != nil {
		return nil, fmt.Errorf("[ConfigError] Unmarshall cookie error: %v", err)
	}

	var jar []*http.Cookie
	for _, cookie := range cjar.Cookies {
		c := &http.Cookie{
			Name:   cookie["Name"],
			Value:  cookie["Value"],
			Path:   cookie["Path"],
			Domain: cookie["Domain"],
		}
		exp, err := strconv.Atoi(cookie["Expires"])
		switch err {
		case nil:
			c.Expires = time.Now().AddDate(10, 0, 0)
		default:
			c.Expires = time.Unix(int64(exp), 0)
		}
		jar = append(jar, c)
	}
	return jar, nil
}

func writeCookieJar(cjFile io.Writer, cookies []*http.Cookie) error {
	b, err := json.Marshal(cookies)
	if err != nil {
		return err
	}
	_, err = cjFile.Write(b)
	return err
}
