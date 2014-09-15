package gumshoe

import (
	"github.com/thoj/go-ircevent"
	"fmt"
	"log"
  "regexp"
  "strings"
	"time"
)

var irc_client *irc.Connection
var irc_enabled = make(chan bool)
var announceLine, episodePattern *regexp.Regexp

func init() {
  // Metrics

  // don't immediately launch the IRC client
	irc_enabled <- false

  // TODO(ryan): make this configurable
  announceLine := regexp.MustCompile("BitMeTV-IRC2RSS: (?P<title>.*?) : (?P<url>.*)")
  episodePattern := regexp.MustCompile("^([\\w\\d\\s.]+)[. ](?:s(\\d{1,2})e(\\d{1,2})|(\\d)x?(\\d{2})|Star.Wars)([. ])")
}

// should this be refactored so that it can reconnect on config changes instead of diconnect and
// connect. TODO(ryan)
func ConnectToTrackerIRC() {
	// Give the connection the configured defaults
	irc_client.KeepAlive = time.Duration(tc.IRC.KeepAlive) * time.Minute
	irc_client.Timeout = time.Duration(tc.IRC.Timeout) * time.Minute
	irc_client.PingFreq = time.Duration(tc.IRC.PingFreq) * time.Minute
	irc_client.Password = tc.IRC.Key
	irc_client.AddCallback("invite", func(e *irc.Event) {
		if strings.Index(e.Raw, tc.IRC.WatchChannel) != -1 {
			irc_client.Join(tc.IRC.WatchChannel)
		}
	})
	irc_client.AddCallback("public", MatchAnnounce)
	var server = fmt.Sprintf("%s:%d", tc.IRC.Server, int(tc.IRC.IRCPort))
	irc_client.Connect(server)
	time.Sleep(60)
	irc_client.SendRawf(tc.IRC.InviteCmd, tc.IRC.Nick, tc.IRC.Key)
}

func MatchAnnounce(e *irc.Event) {
  aMatch := announceLine.FindStringSubmatch(e.Raw)
  if aMatch != nil {
    eMatch := episodePattern.FindStringSubmatch(aMatch[1])
    if eMatch != nil {
      err := IsNewEpisode(eMatch)
      if err == nil {
        log.Println("This is where we would pick up the new episode.")
        // go RetrieveEpisode(aMatch[2])
        return
      }
      log.Println(err)
    }
  }
}

func EnableIRC() {
	for {
		run := <-irc_enabled
		if run {
      log.Println("starting up IRC client.")
			ConnectToTrackerIRC()
		}
	}
}

func DisableIRC() {
	for {
		run := <-irc_enabled
		if !run {
      log.Println("stopping the IRC client.")
			irc_client.Disconnect()
		}
	}
}

func StartIRC() {
  irc_client = irc.IRC(tc.IRC.Nick, tc.IRC.Nick)
	for {
		go EnableIRC()
		go DisableIRC()
		// go WatchIRCConfig(signals)
		// go UpdateLog()
		if tc.Operations.WatchMethod == "irc" {
			irc_enabled <- true
		} else {
			irc_enabled <- false
		}
	}
}
