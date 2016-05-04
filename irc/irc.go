package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/thoj/go-ircevent"
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
    isNew := ep.IsNewEpisode()
    if !isNew {
      PrintDebugln("We already have this episode.")
      return
    }
    if !ep.ValidEpisodeQuality(aMatch[1]) {
      PrintDebugf("Episode %s isn't the right quality.\n", aMatch[1])
      return
    }

    ff, err := NewFileFetch(aMatch[2])
    if err != nil {
      log.Println(err)
    }
    err = ff.RetrieveEpisode()
    if err != nil {
      log.Printf("FAIL: episode not retrieved: %s\n", err)
    }

	  err = ep.AddEpisode()
		if err != nil {
			log.Printf("Episode is downloading, but didn't update the db: %s\n", err)
		}
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
