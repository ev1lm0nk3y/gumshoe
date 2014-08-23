package config_parser

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

// The primary structure holding the config data.

type IMDBConfig struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	Uid  int    `json:"uid"`
}

type IRCChannel struct {
	Nick         string `json:"nick"`
	Key          string `json:"key"`
	Server       string `json:"server"`
	InviteCmd    string `json:"invite_cmd"`
	WatchChannel string `json:"watch_channel"`
	KeepAlive    int    `json:"keep_alive"`
	PingFreq     int    `json:"ping_frequency"`
	IRCPort      int    `json:"irc_port"`
	Timeout      int    `json:"timeout"`
}

type RSSChannel struct {
	FeedURI      string `json:"feed"`
	Passkey      string `json:"passkey"`
	Uid          string `json:"rss_uid"`
	RssTtl       int    `json:"rss_ttl"`
	UseServerTtl bool   `json:"use_server_ttl"`
}

type Operations struct {
	EnableLogging bool   `json:"enable_logging"`
	EnableWeb     bool   `json:"enable_web"`
	HttpPort      string `json:"http_port"`
	LogLevel      string `json:"log_level"`
	UseIMDB       bool   `json:"use_imdb_watchlist"`
	WatchMethod   string `json:"watch_method"`
}

type TrackerConfig struct {
	Cookiejar    map[string]string `json:"cookiejar"`
	Files        map[string]string `json:"file_options"`
	IMDB         IMDBConfig
	IRC          IRCChannel
	Operations   Operations
	RSS          RSSChannel
	Tracker      map[string]interface{} `json:"tracker"`
	LastModified int                    `json:"last_modified"`
}

// Functions (add more comments!!!)
func NewTrackerConfig() *TrackerConfig {
	return &TrackerConfig{}
}

func (tc *TrackerConfig) LoadGumshoeConfig(cfg_file string) error {
	if err := tc.ProcessBaseConfigJSON(cfg_file); err == nil {
		return err
	}
	log.Println("Error with config file %s: %s", cfg_file)
	log.Println("Using basic template.")
	return tc.ProcessBaseConfigJSON("/home/ryan/Development/bitme-get/configs/bitme-get.cfg")
}

func (tc *TrackerConfig) ProcessBaseConfigJSON(config_json string) error {
	if cfg_buf, err := ioutil.ReadFile(config_json); err != nil {
		return err
	} else {
		if err := json.Unmarshal(cfg_buf, &tc); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TrackerConfig) LoadPatterns() []string {
				return []string
}
