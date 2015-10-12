package gumshoe

import (
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thoj/go-ircevent"
)

var (
	// Regexp for messages from IRC channel announcing something to do something about
	announceLine *regexp.Regexp
  // Regexp to determine if the announce regexp matches a known daily episode structure
  //dailyPattern *regexp.Regexp
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

func connectToTracker() {
	log.Printf("Connection to %s:%d commencing.\n", tc.IRC.Server, tc.IRC.Port)
	server := tc.IRC.Server + ":" + strconv.Itoa(tc.IRC.Port)
	if err := ircClient.Connect(server); err != nil {
		IRCConfigError <- err
	}
	if ircClient.Connected() {
		ircStatus.Set("Connected")
		ircConnectTimestamp.Set(time.Now().Unix())
		registerNick()
		// give the server a chance to see the user before attempting to watch the IRC channel.
		time.Sleep(5 * time.Second)
		watchIRCChannel()
	}
}

func watchIRCChannel() {
	if tc.IRC.InviteCmd != "" {
		invite := strings.Replace(tc.IRC.InviteCmd, "%n%", tc.IRC.Nick, -1)
		invite = strings.Replace(invite, "%k%", tc.IRC.Key, -1)
		PrintDebugf("Sending invite to %s: %s\n", tc.IRC.ChannelOwner, invite)
		ircStatus.Set("Requesting Invite")
		ircClient.Privmsgf(tc.IRC.ChannelOwner, invite)
	} else {
		if tc.IRC.WatchChannel != "" {
			log.Printf("Joining channel %s", tc.IRC.WatchChannel)
			ircClient.Join(tc.IRC.WatchChannel)
		}
	}
}

func registerNick() {
	if tc.IRC.Nick == "" {
		PrintDebugln("No nickname set. IRC will not work properly.")
		return
	}
	ircClient.Nick(tc.IRC.Nick)
	if !tc.IRC.Registered {
		ircClient.Privmsgf("nickserv", "register %s %s", tc.IRC.Key, tc.Operations.Email)
	}
	if ircClient.Connected() && tc.IRC.Registered {
		PrintDebugln("identifying to nickserv")
		ircClient.Privmsgf("nickserv", "identify %s", tc.IRC.Key)
	}
}

func msgToUser(e *irc.Event) {
	PrintDebugf("msgToUser: %s", e.Message())
	msg := e.Message()
	if e.User == "NickServ" {
		if strings.Contains(msg, "isn't") || strings.Contains(msg, "incorrect") {
			IRCConfigError <- errors.New(msg)
		} else if strings.Contains(msg, "registered") {
			ircStatus.Set("Nick Ready")
		} else if strings.Contains(msg, "assword") {
			ircStatus.Set("Nick Registered")
		}
	} else {
    PrintDebugf("msgToUser: checking message for show announcement.")
    go matchAnnounce(e)
  }
}

func matchAnnounce(e *irc.Event) {
	PrintDebugf("matchAnnounce: %s\n", e.Message())
	metricUpdate <- time.Now().Unix()
	aMatch := announceLine.FindStringSubmatch(e.Message())
	if aMatch != nil {
    PrintDebugln("matchAnnounce: IRC message is a valid announce line.")
    ep, err := ParseTorrentString(aMatch[1])
    if err != nil {
      PrintDebugf("Error parsing string: %s\n", err)
      return
    }
		// Want to make sure we don't attempt to read/write to the Db at the same
		// time, so during the next call, we block all other updates.
		checkDBLock <- 1
    if ep.IsNewEpisode() && ep.ValidEpisodeQuality(aMatch[1]) {
      AddEpisodeToQueue(aMatch[2])
      err := ep.AddEpisode()
      if err != nil {
        log.Printf("Episode is downloading, but didn't update the db: %s\n", err)
      }
    }
    <-checkDBLock
	}
}

func handleInvite(e *irc.Event) {
	PrintDebugf("handleInvite: %s\n", e.Message())
	if tc.IRC.WatchChannel == "" {
		log.Println("Ignoring invite event because no channels are tracked.")
		return
	}
	PrintDebugf("Handling IRC invite event: %s", e.Message())
	c := e.Connection
	if strings.Index(e.Message(), tc.IRC.WatchChannel) != -1 {
		PrintDebugln("IRC channel invitation successful. Joining Now.")
		c.Join(tc.IRC.WatchChannel)
		ircStatus.Set("Watching Channel")
		if c.Log != nil {
			c.Log.SetPrefix(tc.IRC.WatchChannel + ": ")
		}
	}
}

func _InitIRC() {
	ircClient = irc.IRC(tc.IRC.Nick, tc.IRC.Nick)

	ircClient.Password = tc.IRC.Key
	ircClient.PingFreq = time.Duration(tc.IRC.PingFreq) * time.Minute

	// Callbacks for various IRC events.
	ircClient.AddCallback("invite", handleInvite)
	ircClient.AddCallback("msg", matchAnnounce)
	ircClient.AddCallback("privmsg", matchAnnounce)

	ar, _ := url.QueryUnescape(tc.IRC.AnnounceRegexp)
	announceLine = regexp.MustCompile(ar)
  //dailyPattern = regexp.MustCompile("([\\\\w\\\\d\\\\s]+)\\\\.(\\\\d{4}\\\\.\\\\d{2}\\\\.\\\\d{2})(.+)\\\\.([1080p|720p|HDTV|hdtv]).+$")
	ircStatus.Set("Ready")
}

func _TrackIRCStatus() {
	for {
		select {
		// Turn on and off the IRC service.
		case e := <-IRCEnabled:
			if e && ircClient == nil {
				_InitIRC()
				if ircClient == nil {
					IRCConfigError <- errors.New("IRC Configuration Issues. Not Connected.")
				}
			} else if e && ircClient != nil && !ircClient.Connected() {
				connectToTracker()
			} else if !e && ircClient.Connected() {
				ircClient.Disconnect()
			}
		// Update the IRC configuration.
		case cfg := <-IRCConfigChanged:
			if cfg {
				if ircClient != nil {
					ircClient.Disconnect()
				}
				if tc.Operations.WatchMethods["irc"] {
					_InitIRC()
					if ircClient == nil {
						IRCConfigError <- errors.New("IRC Configuration Issues. Not Connected.")
					}
					connectToTracker()
				}
			}
		// Updates the lastest timestamp metric.
		case ts := <-metricUpdate:
			last, _ := strconv.Atoi(ircUpdateTimestamp.String())
			if ts > int64(last) {
				ircUpdateTimestamp.Set(ts)
			}
		case err := <-IRCConfigError:
			if err == nil {
				continue
			}
			log.Println(err)
			ircStatus.Set(fmt.Sprintf("Config Error: %s\n", err))
			ircClient.Disconnect()
			return
		}
	}
}

func StartIRC() {
	_InitIRC()
	connectToTracker()

	go _TrackIRCStatus()
	IRCEnabled <- tc.Operations.WatchMethods["irc"]
	IRCConfigChanged <- false
	IRCConfigError <- nil
	metricUpdate <- 0
}
