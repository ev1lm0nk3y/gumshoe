package gumshoe

import (
	"log"
  "os"
  //"path/filepath"
	"regexp"
	"strings"
  "strconv"
	"time"

	"github.com/thoj/go-ircevent"
)
var (
  announceLine *regexp.Regexp
  episodePattern *regexp.Regexp
  watchChannel string
  //watchingChannel chan bool
)

func init() {
	// Metrics

	// TODO(ryan): make this configurable
	announceLine = regexp.MustCompile("BitMeTV-IRC2RSS: (?P<title>.*?) : (?P<url>.*)")
	episodePattern = regexp.MustCompile("^([\\w\\d\\s.]+)[. ](?:s(\\d{1,2})e(\\d{1,2})|(\\d)x?(\\d{2})|Star.Wars)([. ])")
}

// should this be refactored so that it can reconnect on config changes instead of diconnect and
// connect. TODO(ryan)
func connectToTracker(tc *TrackerConfig, c *irc.Connection) error {
  server := tc.IRC.Server + ":" + strconv.Itoa(tc.IRC.IRCPort)
  if connerr := c.Connect(server); connerr != nil {
    return connerr
  }
  if tc.IRC.NeedInvite {
    c.Nick(tc.IRC.Nick)
    if c.Debug {
      log.Println("Sleeping for 5s before requesting the invite.")
    }
    time.Sleep(5 * time.Second)
    if tc.IRC.ChannelOwner != "" {
      if c.Debug {
        log.Printf("sending invite message to %s", tc.IRC.ChannelOwner)
      }
      c.Privmsgf(tc.IRC.ChannelOwner, "!invite %s %s", tc.IRC.Nick, tc.IRC.Key)
    }
  } else {
    if (tc.IRC.WatchChannel != "") {
      log.Printf("Joining channel %s", tc.IRC.WatchChannel)
      c.Join(tc.IRC.WatchChannel)
    }
  }
  return nil
}

func matchAnnounce(e *irc.Event) {
	aMatch := announceLine.FindStringSubmatch(e.Message())
	if aMatch != nil {
		eMatch := episodePattern.FindStringSubmatch(aMatch[1])
		if eMatch != nil {
			err := IsNewEpisode(eMatch)
			if err != nil {
        log.Println(err)
      } else {
				log.Println("This is where we would pick up the new episode.")
				// go RetrieveEpisode(aMatch[2])
			}
		} else {
      log.Println("Not an episode match.")
    }
	} else {
    log.Println("Not an episode announcement. Basicaly garbage.")
  }
}

func handleInvite(e *irc.Event) {
  if watchChannel == "" {
    log.Println("Ignoring invite event because no channel has been designated to watch.")
    return
  }
  if e.Connection.Debug {
    log.Printf("Handling IRC invite event: %s", e.Message())
  }
  c := e.Connection
	if strings.Index(e.Message(), watchChannel) != -1 {
    if e.Connection.Debug {
      log.Printf("We have been invited to %s. Joining Now.", watchChannel)
    }
		c.Join(watchChannel)
    if c.Log != nil {
      c.Log.SetPrefix(watchChannel + ": ")
    }
	}
}

// StartIRC kick off the IRC client
func StartIRC(tc *TrackerConfig) (*irc.Connection, error) {
  ircClient := irc.IRC(tc.IRC.Nick, tc.IRC.Nick)
  ircLog, err := os.OpenFile("irc.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
  if err == nil {
    ircClient.Log = log.New(ircLog, "", log.LstdFlags)
  } else {
    log.Println("Unable to open log file for IRC logging. Writing IRC logs to STDOUT")
  }
  watchChannel = tc.IRC.WatchChannel

  ircClient.Password = tc.IRC.Key
  ircClient.PingFreq = 15 * time.Minute
  ircClient.Debug = tc.IRC.Debug

  // Callbacks for various IRC events.
	ircClient.AddCallback("invite", handleInvite)
	ircClient.AddCallback("msg", matchAnnounce)
  ircClient.AddCallback("privmsg", matchAnnounce)

  // Now make the final connection to the IRC tracker with a timeout
  /*timeout := time.AfterFunc(2 * time.Minute,
    func() {
      ircClient.Disconnect()
      return errors.New("IRC connection timed out.")
    }(error)
  )
  */
  err = connectToTracker(tc, ircClient)
  return ircClient, err
}
