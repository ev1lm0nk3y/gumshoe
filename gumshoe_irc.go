package gumshoe

import(
	"fmt"
	"github.com/thoj/go-ircevent/irc"
	"log"
	"time"
)

// Metrics
func init() {
	patterns := config_parser.LoadPatterns()
	// parse and start metrics

	// irc_client is the global irc connection manager, initialize it as stopped
	var irc_client = make(irc.Connection)
	irc_client.stopped = true

	// enabled lets us know if we should run the irc client
	var enabled = make(chan bool)
	enabled <- false
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
	irc_client.AddCallback("public", EpisodePatternMatch)
	var server = fmt.Sprintf("%s:%d", tc.IRCChannel.Server, tc.IRCChannel.IRCPort.(int32))
	irc_client.Connect(server)
	time.sleep(60)
	irc_client.SendRawf(tc.IRCChannel.InviteCmd, tc.IRCChannel.Nick, tc.IRCChannel.Key)
}

func EpisodePatternMatch(e *Event) {
	if ap := patterns.AnnounceLine.FindStringSubmatch(string.toLower(e.Raw)); ap != nil {
		for p := range patterns.Shows {
			if match, _ := regexp.MatchString(p.Title, ap[1]); match {
				log.Println("Episode match found. %s", e.Raw)
				if p.EpisodeOnly {
					if patterns.EpisodePattern.MatchString(ap[1]) {
						log.Println("This is an episode.")
					} else {
						log.Println("Not interested.")
						return
					}
				}
				if !patterns.ExcludePatterns.MatchString(ap[1]) {
					if !downloader.CheckPreviousEpisodes(ap[1]) {
						log.Println("Full match, grabbing.")
						downloader.GetEpisode(ap[2])
					}
				}
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
