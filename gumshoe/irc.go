package gumshoe

import (
	"errors"
	"expvar"
	"log"
	//"os"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thoj/go-ircevent"
)

var (
	announceLine        *regexp.Regexp
	episodePattern      *regexp.Regexp
	ircConnectTimestamp = expvar.NewInt("irc_connect_timestamp")
	ircUpdateTimestamp  = expvar.NewInt("irc_last_update_timestamp")
  ircStatus           = expvar.NewString("irc_status")
	ircClient           *irc.Connection
	metricUpdate        = make(chan int64)
	checkDBLock         = make(chan int)
	IRCEnabled          = make(chan bool)
	IRCConfigChanged    = make(chan bool)
	IRCConfigError      = make(chan error)
  ErrNoUser           = errors.New("No such user online or registered.")
  ErrNotRecognized    = errors.New("user not recognized as nickname's owner.")
  ErrListOnly         = errors.New("user recognized as owner via access list only.")
)

func connectToTracker() {
	log.Printf("Connection to %s:%d commencing. ", tc.IRC.Server, tc.IRC.Port)
	server := tc.IRC.Server + ":" + strconv.Itoa(tc.IRC.Port)
	if err := ircClient.Connect(server); err != nil {
		IRCConfigError <- err
	}
  ircStatus.Set("Connected")
	ircConnectTimestamp.Set(time.Now().Unix())
  registerNick()
  watchIRCChannel()
}

func watchIRCChannel() {
	if tc.IRC.InviteCmd != "" {
    invite := strings.Replace(tc.IRC.InviteCmd, "%n%", tc.IRC.Nick, -1)
    invite = strings.Replace(invite, "%k%", tc.IRC.Key, -1)
    PrintDebugf("Sending invite: %s\n", invite)
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
  if tc.IRC.Nick ==  "" {
    PrintDebugln("No nickname set. IRC will not work properly.")
    return
  }
  ircClient.Nick(tc.IRC.Nick)
  while !tc.IRC.Registered {
    err := <-IRCConfigError

    ircClient.Privmsgf("nickserv", "register %s %s", tc.IRC.Key, tc.Operations.Email)
  }
  ircClient.Privmsgf("nickserv", "identify %s", tc.IRC.Key)
  PrintDebugln("IRC nick ready for use")
}

func msgToUser(e *irc.Event) {
  if e.User == "NickServ" {
    if strings.Contains(e.Message, "isn't") || strings.Contains(e.Message, "incorrect") {
      IRCConfigError<- errors.New(e.Message)
      return
    }
    if strings.Contains("registered") {
      ircStatus.Set("Nick Ready")
    }
    if strings.Contains(e.Message, "Password" {
      ircStatus.Set("Nick Registered")
    }
  }
}

func matchAnnounce(e *irc.Event) {
	metricUpdate <- time.Now().Unix()
	aMatch := announceLine.FindStringSubmatch(e.Message())
	if aMatch != nil {
		eMatch := episodePattern.FindStringSubmatch(aMatch[1])
		if eMatch != nil {
			// Want to make sure we don't attempt to read/write to the Db at the same
			// time, so during the next call, we block all other updates.
			checkDBLock <- 1
			err := IsNewEpisode(eMatch)
			<-checkDBLock
			if err != nil {
				return
			}
			AddEpisodeToQueue(aMatch[2])
		}
	}
}

func handleInvite(e *irc.Event) {
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
	ircClient.AddCallback("privmsg", msgToUser)

	ar, _ := url.QueryUnescape(tc.IRC.AnnounceRegexp)
	announceLine = regexp.MustCompile(ar)
	er, _ := url.QueryUnescape(tc.IRC.EpisodeRegexp)
	episodePattern = regexp.MustCompile(er)
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
