package gumshoe

import (
	"github.com/thoj/go-ircevent"
	"fmt"
	// "log"
  "regexp"
	"time"
)

// Metrics
func init() {
  // Metrics

	// irc_client is the global irc connection manager, initialize it as stopped
	var irc_client *irc.Connection

	// enabled lets us know if we should run the irc client
	var enabled = make(chan bool)
	enabled <- false

  // How are the episodes announced
  // TODO(ryan): make this configurable
  announceLine := regexp.MustCompile("BitMeTV-IRC2RSS: (?P<title>.*?) : (?P<url>.*)")
  episodePattern := regexp.MustCompile("^([\\w\\d\\s.]+)[. ](?:s(\\d{1,2})e(\\d{1,2})|(\\d)x?(\\d{2})|Star.Wars)([. ])")
}

// should this be refactored so that it can reconnect on config changes instead of diconnect and
// connect. TODO(ryan)
func ConnectToTrackerIRC(irc_client *irc.Connection) {
	// Give the connection the configured defaults
	irc_client.KeepAlive = time.Duration(tc.IRC.KeepAlive) * time.Minute
	irc_client.Timeout = time.Duration(tc.IRC.Timeout) * time.Minute
	irc_client.PingFreq = time.Duration(tc.IRC.PingFreq) * time.Minute
	irc_client.Password = tc.IRC.Key
	irc_client.AddCallback("invite", func(e *irc.Event) {
		if string.Index(e.Raw, tc.IRC.WatchChannel) != -1 {
			irc_client.Join(tc.IRC.WatchChannel)
		}
	})
	irc_client.AddCallback("public", MatchAnnounce)
	var server = fmt.Sprintf("%s:%d", tc.IRC.Server, tc.IRC.IRCPort.(int32))
	irc_client.Connect(server)
	time.Sleep(60)
	irc_client.SendRawf(tc.IRC.InviteCmd, tc.IRC.Nick, tc.IRC.Key)
}

func MatchAnnounce(e *irc.Event) {
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
		if run && !irc_client.stopped {
			// some log lines here and stauts updates
			ConnectToTrackerIRC()
		}
	}
}

func DisableIRC() {
	for {
		run := <-enabled
		if !run && irc_client.stopped {
			// some log line and stauts updates
			irc_client.Disconnect()
		}
	}
}

func StartIRC() {
	for {
		go EnableIRC()
		go DisableIRC()
		// go WatchIRCConfig(signals)
		// go UpdateLog()
		if tc.Operations.WatchMethod == "irc" {
			enabled <- true
		} else {
			enabled <- false
		}
	}
}
