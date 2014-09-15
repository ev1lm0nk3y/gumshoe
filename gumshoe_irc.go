package gumshoe

import (
	"fmt"
	"github.com/thoj/go-ircevent/irc"
	"log"
  "regexp"
	"time"
)

// Metrics
func init() {
  // Metrics

	// irc_client is the global irc connection manager, initialize it as stopped
	var irc_client = make(irc.Connection)
	irc_client.stopped = true

	// enabled lets us know if we should run the irc client
	var enabled = make(chan bool)
	enabled <- false

  // How are the episodes announced
  // TODO(ryan): make this configurable
  announceLine := regexp.MustCompile("BitMeTV-IRC2RSS: (?P<title>.*?) : (?P<url>.*)")
  episodePattern := regexp.MustCompile("^([\w\d\s.]+)[. ](?:s(\d{1,2})e(\d{1,2})|(\d)x?(\d{2})|Star.Wars)([. ])")
}

// should this be refactored so that it can reconnect on config changes instead of diconnect and
// connect. TODO(ryan)
func ConnectToTrackerIRC(tc *config_parser.TrackerConfig) {
	irc_client := irc.IRC(tc.IRCChannel.Nick, tc.IRCChannel.Nick)
	// Give the connection the configured defaults
	irc_client.KeepAlive = tc.IRCChannel.KeepAlive * time.Minute
	irc_client.Timeout = tc.IRCChannel.Timeout * time.Minute
	irc_client.PingFreq = tc.IRCChannel.PingFrequency * time.Minute
	irc_client.Password = tc.IRCChannel.Key
	irc_client.AddCallback("invite", func(e *Event) {
		if string.Index(e.Raw, tc.IRCChannel.WatchChannel) != -1 {
			irc_client.Join(tc.IRCChannel.WatchChannel)
		}
	})
	irc_client.AddCallback("public", MatchAnnounce)
	var server = fmt.Sprintf("%s:%d", tc.IRCChannel.Server, tc.IRCChannel.IRCPort.(int32))
	irc_client.Connect(server)
	time.sleep(60)
	irc_client.SendRawf(tc.IRCChannel.InviteCmd, tc.IRCChannel.Nick, tc.IRCChannel.Key)
}

func MatchAnnounce(e *Event) {
  aMatch := announceLine.FindStringSubmatch(e.Raw)
  if aMatch != nil {
    eMatch := episodePattern.FindStringSubmatch(aMatch[1])
    if eMatch != nil {
      if watcher.IsNewEpisode(eMatch) {
        go downloader.RetrieveEpisode(aMatch[2])
      }
    }
  }
}

func EnableIRC() {
	for {
		run := <-enabled
		if run && irc_client.stopped {
			// some log lines here and stauts updates
			ConnectToTrackerIRC(config)
		}
	}
}

func DisableIRC() {
	for {
		run := <-enabled
		if !run && !irc_client.stopped {
			// some log line and stauts updates
			irc_client.Disconnect()
		}
	}
}

func startIRC(signals <-chan GumshoeSignals, config *config_parser.TrackerConfig) {
	for {
		go EnableIRC()
		go DisableIRC()
		// go WatchIRCConfig(signals)
		// go UpdateLog()
		if config.Operations.WatchMethod == "irc" {
			enabled <- true
		} else {
			enabled <- false
		}
	}
}
