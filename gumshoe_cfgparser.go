package gumshoe

import (
	"encoding/json"
  "errors"
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

func NewTrackerConfig() *TrackerConfig {
	return &TrackerConfig{}
}

func (tc *TrackerConfig) LoadGumshoeConfig(cfgFile string) error {
	if err := tc.ProcessGumshoeJSON(cfgFile); err != nil {
		log.Println("Error with config file ", cfgFile, ": ", err)
		log.Println("Using basic template.")
		return tc.ProcessGumshoeJSON("cfg/gumshoe_config.json")
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
	// config <- true
	return nil
}

func (tc *TrackerConfig) WriteGumshoeConfig(update []byte) error {
	err := json.Unmarshal(update, &tc)
  if err == nil {
		var gCfg []byte
		gCfg, err := json.MarshalIndent(&tc, "", "  ")
    if err == nil {
			return ioutil.WriteFile(tc.Files["base_dir"]+"gumshoe_config.json",
				gCfg, 0655)
		}
		return err
	}
	return err
}

// TV shows that should be downloaded.
var allShows *Shows

type Show struct {
	Title    string   `json:"title"`
	Quality  []string `json:"quality"`
	Episodal bool     `json:"episodes"`
}

type Shows struct {
	TVShows []Show `json:"tv shows"`
}

func NewShowsConfig() *Shows {
	return &Shows{}
}

//func (S *Shows) AddShow(s Show) {
//	append(S.TVShows, s)
//}

func (S *Shows) GetShow(title string) (int, *Show, error) {
	for x := range S.TVShows {
		if S.TVShows[x].Title == title {
			return x, &S.TVShows[x], nil
		}
	}
	return -1, nil, errors.New("Not found")
}

//func (S *Shows) RemoveShow(title string) error {
//	var index int
//	if index, _, err := S.GetShow(title); err != nil {
//		return err
//	}
//	firstSlice := S.TVShows[:index-1]
//  for x := range S.TVShows[index+1:] {
//    append(firstSlice, x)
//  }
//	S.TVShows = firstSlice
//	return nil
//}

func (S *Shows) LoadShows() (int, error) {
	sCfg, err := ioutil.ReadFile(tc.Files["shows"])
  if err != nil {
		log.Println("No show file found. Will just use a blank one.")
	}
	if err := json.Unmarshal(sCfg, &S); err != nil {
		// TODO(ryan): log error message
		return -1, err
	}
	return len(S.TVShows), nil
}

func (S *Shows) WriteShows() (int, error) {
	sCfg, err := json.MarshalIndent(&S, "", "  ")
  if err == nil {
		// TODO(ryan): log success writing things (woo!)
		if err := ioutil.WriteFile(tc.Files["shows"], sCfg, 0666); err == nil {
			// TODO(ryan): log more stuff here too.
			return len(S.TVShows), nil
		}
	}
	return -1, err
}
