package irc

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

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/ev1lm0nk3y/gumshoe/misc"
	irc_client "github.com/thoj/go-ircevent"
)

const (
	maxLogLines = 100
)

var (
	tc *config.TrackerConfig
	// Time, in ms, when the connection to the IRC server was established
	ircConnectTimestamp = expvar.NewInt("irc_connect_timestamp")
	// Time, in ms, when the channel was last updated
	ircUpdateTimestamp = expvar.NewInt("irc_last_update_timestamp")
	// String relating to the current state of the IRC watcher
	ircStatus = expvar.NewString("irc_status")
	// IRC client object
	ircClient *irc_client.Connection
	// channel that gets timestamp updates for ircUpdateTimestamp in order to ensure we write only the most recent timestamp into that exported variable.
	metricUpdate = make(chan int64)
	// Regexp for messages from IRC channel announcing something to do something about
	announceLine *regexp.Regexp
	// All messages from the IRC server will be recorded to view later.
	ircLog  = make(chan string)
	logLines = []string{}
)

type IRCControlChannel struct {
	// Channel that is used to turn on and off the IRC watcher.
	IRCEnabled chan bool
	// Channel to signify if the IRC config has changed. Changes will restart the IRC watcher.
	IRCConfigChanged chan bool
	// Channel that collects all IRC errors and will disconnect the IRC watcher if it encounters one.
	IRCConfigError chan error
	// Channel for announceline matches
	IRCAnnounceMatch chan []string
	// Channel for letting the sign in process know when the nick has been registered.
	IRCNickRegistered chan bool
}

var icc = &IRCControlChannel{}

func connectToTracker() {
	log.Printf("Connection to %s:%d commencing.\n", tc.IRC.Server, tc.IRC.Port)
	server := tc.IRC.Server + ":" + strconv.Itoa(tc.IRC.Port)
	if err := ircClient.Connect(server); err != nil {
		misc.PrintDebug(err)
		icc.IRCConfigError <- err
		return
	}

	ircStatus.Set("Connected")
	ircConnectTimestamp.Set(time.Now().Unix())
	if err := registerNick(); err != nil {
		icc.IRCConfigError <- err
		ircStatus.Set("Not Connected")
		ircConnectTimestamp.Set(0)
		return
	}
	// give the server a chance to see the user before attempting to watch the IRC channel.
	time.Sleep(5 * time.Second)
	watchIRCChannel()
}

func watchIRCChannel() {
	if tc.IRC.InviteCmd != "" {
		invite := strings.Replace(tc.IRC.InviteCmd, "%n%", tc.IRC.Nick, -1)
		invite = strings.Replace(invite, "%k%", tc.IRC.Key, -1)
		ircStatus.Set("Requesting Invite")
		ircClient.Privmsgf(tc.IRC.ChannelOwner, invite)
	}
	if tc.IRC.WatchChannel != "" {
		log.Printf("Joining channel %s", tc.IRC.WatchChannel)
		ircClient.Join(tc.IRC.WatchChannel)
	}
}

func registerNick() error {
	if tc.IRC.Nick == "" || tc.IRC.Key == "" {
		return fmt.Errorf("No nickname or IRC key set. %s not fully connected", tc.IRC.Server)
	}

	ircClient.Nick(tc.IRC.Nick)
	if !tc.IRC.Registered {
		ircClient.Privmsgf("NickServ", "register %s %s", tc.IRC.Key, tc.Operations.Email)

		// Wait 60s until the nick has been registered to ensure we can continue.
		regTimer := time.AfterFunc(time.Second*60, func() {
			close(icc.IRCNickRegistered)
			icc.IRCConfigError<- fmt.Errorf("timeout after waiting to register IRC nickname %s", tc.IRC.Nick)
		})

		<-icc.IRCNickRegistered
		regTimer.Stop()
	}

	ircClient.Privmsgf("NickServ", "identify %s", tc.IRC.Key)
	return nil
}

func msgToUser(e *irc_client.Event) {
	ircLog <- e.Message()
	go matchAnnounce(e)
	// Check for other messages that I might want to act upon.
	go func() {
		msg := e.Message()
		switch {
		case strings.Contains(msg, "isn't") || strings.Contains(msg, "incorrect"):
			icc.IRCConfigError <- errors.New(msg)
		case strings.Contains(msg, "registered"):
			ircStatus.Set("Nick Ready")
		case strings.Contains(msg, "assword"):
			icc.IRCNickRegistered <- true
			ircStatus.Set("Nick Registered")
			tc.IRC.Registered = true
		}
	}()
}

func matchAnnounce(e *irc_client.Event) {
	ircLog <- e.Message()
	metricUpdate <- time.Now().Unix()
	aMatch := announceLine.FindStringSubmatch(e.Message())
	if aMatch != nil {
		icc.IRCAnnounceMatch <- aMatch
	}
}

func handleInvite(e *irc_client.Event) {
	ircLog <- e.Message()
	if tc.IRC.WatchChannel == "" {
		log.Println("Ignoring invite event because no channels are tracked.")
		return
	}
	go func() {
		c := e.Connection
		if strings.Index(e.Message(), tc.IRC.WatchChannel) != -1 {
			misc.PrintDebugln("IRC channel invitation successful. Joining Now.")
			c.Join(tc.IRC.WatchChannel)
			ircStatus.Set("Watching Channel")
			if c.Log != nil {
				c.Log.SetPrefix(tc.IRC.WatchChannel + ": ")
			}
		}
	}()
}

func initIRC() {
	ircClient = irc_client.IRC(tc.IRC.Nick, tc.IRC.Nick)

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

func trackIRCStatus() {
	for {
		select {
		// Turn on and off the IRC service.
		case e := <-icc.IRCEnabled:
			if e && ircClient == nil {
				initIRC()
				if ircClient == nil {
					icc.IRCConfigError <- errors.New("IRC Configuration Issues. Not Connected.")
				}
			} else if e && ircClient != nil && !ircClient.Connected() {
				connectToTracker()
			} else if !e && ircClient.Connected() {
				ircClient.Disconnect()
			}
		// Update the IRC configuration.
		case cfg := <-icc.IRCConfigChanged:
			if cfg {
				if ircClient != nil {
					ircClient.Disconnect()
				}
				initIRC()
				if ircClient == nil {
					icc.IRCConfigError <- errors.New("IRC Configuration Issues. Not Connected.")
				}
				connectToTracker()
			}
		// Updates the lastest timestamp metric.
		case ts := <-metricUpdate:
			last, _ := strconv.Atoi(ircUpdateTimestamp.String())
			if ts > int64(last) {
				ircUpdateTimestamp.Set(ts)
			}
		case err := <-icc.IRCConfigError:
			if err == nil {
				continue
			}
			ircStatus.Set(fmt.Sprintf("Config Error: %s\n", err))
			ircClient.Disconnect()
			return
		case log := <-ircLog:
			if len(logLines) == maxLogLines {
				logLines = logLines[1:]
			}
			logLines = append(logLines, log)
		}
	}
}

func GetIrcLogs() []string {
	return logLines
}

func StartIRC(cfg *config.TrackerConfig) *IRCControlChannel {
	tc = cfg

	initIRC()
	go connectToTracker()
	go trackIRCStatus()

	metricUpdate <- 0
	return icc
}
