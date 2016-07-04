// Package irc connects to a tracker's irc channel and parses messages to see
// if they match a given regex. It can be controlled once via a set of channels
// once started.
//
// Example:
//   ic := irc.InitRC(my_irc_config)
//   err = ic.StartIRC()
//   select {
//   case err = <-ic.IRCError:
//     <do something>
//   case log = <-ic.IRCLog:
//     <do something>
//   }
package irc

import (
	"expvar"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/thoj/go-ircevent"
)

const (
	maxLogLines = 100
	NICK_REG    = 0
	NICK_AUTH   = 1

	IRC_CONNECTED      = "Connected"
	IRC_DISCONNECTED   = "Disconnected"
	IRC_NICK_REG       = "Nick Registration"
	IRC_PENDING_INVITE = "Invite Sent"
	IRC_STOPPED        = "Stopped"
	IRC_WATCHING       = "Watching Channel"
)

var (
	ircConnectTimestamp = expvar.NewInt("irc_connect_timestamp")
	ircUpdateTimestamp  = expvar.NewInt("irc_last_update_timestamp")
	ircStatus           = expvar.NewString("irc_status")
	metricUpdate        = make(chan int64)
	announceLine        *regexp.Regexp
	regStep             = make(chan int)
	reM                 sync.RWMutex
	icM                 sync.RWMutex
)

// IrcControlChannel is made up of channels that help pass messages between
// here and your main program.
type IrcControlChannel struct {
	IRCEnabled       chan bool     // Boolean to turn on or off the IRC client
	IRCConfigChanged chan bool     // Pass new IRC configuration here to update your client
	IRCError         chan error    // Errors and warnings will be posted here for you to deal with
	IRCAnnounceMatch chan []string // When the regex matches, this channel will be updated
	IRCLog           chan string   // IRC logs are exported on this channel
}

type ircClient struct {
	logLines []string
	config   *config.IRCChannel

	*irc.Connection
	*IrcControlChannel
}

// Logger is a reference the the GetIrcLogs function. Can be thrown around to
// different processes.
type Logger func() []string

// InitIrc lays the foundation for communication with an IRC server and returns
// you a client to work with. The next step is to call StartIRC.
// Errors will be returned if your announce line regexp doesn't parse.
func InitIrc(c *config.IRCChannel) (*ircClient, error) {
	ircConn := irc.IRC(c.Nick, c.Key)
	icc := &IrcControlChannel{}
	icc.IRCError = ircConn.ErrorChan()

	icM.Lock()
	defer icM.Unlock()
	ic := &ircClient{
		[]string{"new log started"},
		c,
		ircConn,
		icc,
	}

	// Callbacks for various IRC events.
	ic.AddCallback("001", ic.handleWelcome)
	ic.AddCallback("invite", ic.handleInvite)
	ic.AddCallback("message", ic.handleMsg)
	ic.AddCallback("msg", ic.handleMsg)
	ic.AddCallback("privmsg", ic.handleMsg)
	ic.AddCallback("notice", ic.handleMsg)

	ic.EnableFullIrcLogs(c.EnableLog)
	err := ic.SetWatchRegexp(c.AnnounceRegexp)

	ircStatus.Set(IRC_STOPPED)
	return ic, err
}

// SetWatchRegexp will make sure your client finds the right messages from you
// IRC server. If a string isn't provided, we will grab it from the client config.
// Will return an error if there is a problem with your regexp.
func (ic *ircClient) SetWatchRegexp(al string) error {
	if al == "" {
		icM.RLock()
		defer icM.RUnlock()
		al = ic.config.AnnounceRegexp
	}

	ar, err := url.QueryUnescape(al)
	if err != nil {
		return err
	}

	reM.Lock()
	defer reM.Unlock()
	announceLine, err = regexp.Compile(ar)
	return err
}

// EnableFullIrcLogs will either report on the log channel all the IRC server
// messages, or will turn it off completely.
func (ic *ircClient) EnableFullIrcLogs(l bool) {
	icM.Lock()
	defer icM.Unlock()
	if ic.config.EnableLog {
		ic.AddCallback("*", func(e *irc.Event) {
			ic.IRCLog <- e.Message()
		})
		return
	}
	ic.ClearCallback("*")
}

// StartIRC will do just that. Will connect to the IRC server, register and
// authenticate you nickname and join the channel you want to watch. Errors on
// connection will be returned.
func (ic *ircClient) StartIRC() error {
	go ic.nickRegistration()
	go ic.trackIRCStatus()
	metricUpdate <- 0
	return ic.connectTracker()
}

// GetIrcLogs will return you a slice of the irc logs, up to the last 100. This
// is useful if you are not watching the IRCLog channel.
func (ic *ircClient) GetIrcLogs() []string {
	icM.RLock()
	defer icM.RUnlock()
	return ic.logLines
}

func (ic *ircClient) connectTracker() error {
	icM.Lock()
	defer icM.Unlock()
	s := net.JoinHostPort(ic.config.Server, strconv.Itoa(ic.config.Port))
	ic.IRCLog <- fmt.Sprintf("Connection to %s commencing.\n", s)
	if err := ic.Connect(s); err != nil {
		return err
	}
	ircStatus.Set(IRC_CONNECTED)
	ircConnectTimestamp.Set(time.Now().Unix())
	return nil
}

func (ic *ircClient) disconnectTracker() {
	icM.Lock()
	defer icM.Unlock()
	ic.Disconnect()
	ircStatus.Set(IRC_DISCONNECTED)
	metricUpdate <- time.Now().Unix()
}

func (ic *ircClient) reconnectTracker() error {
	ic.disconnectTracker()
	go ic.nickRegistration()
	ic.EnableFullIrcLogs(ic.config.EnableLog)
	if err := ic.SetWatchRegexp(ic.config.AnnounceRegexp); err != nil {
		return err
	}
	return ic.connectTracker()
}

func (ic *ircClient) nickRegistration() {
	for {
		rs := <-regStep
		switch {
		case rs == NICK_REG:
			ic.Privmsgf("NickServ", "identify %s", ic.config.Key)
		case rs == NICK_AUTH:
			go ic.watchChannel()
			return
		}
	}
}

func (ic *ircClient) sendInvite() error {
	icM.RLock()
	defer icM.RUnlock()
	if ic.config.InviteCmd == "" {
		return fmt.Errorf("No invite command.")
	}
	invite := strings.Replace(ic.config.InviteCmd, "%nick%", ic.config.Nick, -1)
	invite = strings.Replace(invite, "%key%", ic.config.Key, -1)
	ircStatus.Set(IRC_PENDING_INVITE)
	ic.Privmsg(ic.config.ChannelOwner, invite)
	return nil
}

func (ic *ircClient) watchChannel() error {
	icM.RLock()
	defer icM.RUnlock()
	if ic.config.WatchChannel == "" {
		return fmt.Errorf("No channel to watch given.")
	}
	ic.Join(ic.config.WatchChannel)
	return nil
}

func (ic *ircClient) matchAnnounce(m string) {
	metricUpdate <- time.Now().Unix()
	reM.RLock()
	defer reM.RUnlock()
	aMatch := announceLine.FindStringSubmatch(m)
	if aMatch != nil {
		ic.IRCAnnounceMatch <- aMatch
	}
}

// Handle IRC messages.
func (ic *ircClient) handleInvite(e *irc.Event) {
	icM.RLock()
	defer icM.RUnlock()
	if ic.config.WatchChannel == "" {
		ic.IRCError <- fmt.Errorf("Ignoring invite event because no channels are tracked.")
		return
	}
	if strings.HasSuffix(e.Message(), ic.config.WatchChannel) {
		ic.Join(ic.config.WatchChannel)
		ircStatus.Set(IRC_WATCHING)
		if ic.Log != nil {
			ic.Log.SetPrefix(fmt.Sprintf("%s:", ic.config.WatchChannel))
		}
	}
}

func (ic *ircClient) handleWelcome(e *irc.Event) {
	icM.RLock()
	defer icM.RUnlock()
	n := ic.GetNick()
	if n == ic.config.Nick {
		regStep <- NICK_REG
		return
	}
	go ic.Nick(ic.config.Nick)
}

func (ic *ircClient) handleMsg(e *irc.Event) {
	go ic.matchAnnounce(e.Message())
	// Check for other messages that I might want to act upon.
	// Mostly this is to control the nick registration process.
	if ircStatus.String() != IRC_WATCHING {
		go func(ic *ircClient, msg string) {
			switch {
			case strings.Contains(msg, "is registered"):
				regStep <- NICK_REG
				ircStatus.Set(IRC_NICK_REG)
			case strings.Contains(msg, "Password accepted"):
				regStep <- NICK_AUTH
				ircStatus.Set(IRC_NICK_REG)
			}
		}(ic, e.Message())
	}
}

func (ic *ircClient) trackIRCStatus() {
	for {
		select {
		// Turn on and off the IRC service.
		case e := <-ic.IRCEnabled:
			if e && !ic.Connected() {
				ic.IRCError <- ic.StartIRC()
			} else {
				ic.disconnectTracker()
			}
		// Update the IRC configuration.
		case c := <-ic.IRCConfigChanged:
			if c {
				ic.IRCLog <- fmt.Sprintf("IRC channel resetting.")
				ic.IRCError <- ic.reconnectTracker()
			}
		// Updates the lastest timestamp metric.
		case ts := <-metricUpdate:
			last, _ := strconv.Atoi(ircUpdateTimestamp.String())
			if ts > int64(last) {
				ircUpdateTimestamp.Set(ts)
			}
		case log := <-ic.IRCLog:
			icM.Lock()
			// Drop the oldest log line.
			if len(ic.logLines) == maxLogLines {
				ic.logLines = ic.logLines[1:]
			}
			ic.logLines = append(ic.logLines, log)
			icM.Unlock()
		}
	}
}
