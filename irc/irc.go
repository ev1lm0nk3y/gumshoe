// Package irc connects to a tracker's irc channel and parses messages to see
// if they match a given regex. It can be controlled once via a set of channels
// once started.
//
// Example:
//   ic := irc.InitRC(my_irc_config)
//   err = ic.Start()
//   select {
//   case err = <-ic.IRCError:
//     <do something>
//   case log = <-ic.IRCLog:
//     <do something>
//   }
package irc

import (
	"bytes"
	"expvar"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/thoj/go-ircevent"
)

type ircState int

const (
	ircStateList ircState = iota
	connected
	disconnected
	nickReg
	nickAuth
	pendingInvite
	stopped
	watching
)

var (
	ircConnectTimestamp = expvar.NewInt("irc_connect_timestamp")
	ircUpdateTimestamp  = expvar.NewInt("irc_last_update_timestamp")
	ircStatus           = expvar.NewString("irc_status")
	metricUpdate        = make(chan int64)

	ircStateMap = map[ircState]string{
		connected:     "Connected",
		disconnected:  "Disconnected",
		nickReg:       "Nick Registration",
		nickAuth:      "Nick Authorization",
		pendingInvite: "Invite Sent",
		stopped:       "Stopped",
		watching:      "Watching Channel",
	}
)

// IrcControlChannel is made up of channels that help pass messages between
// here and your main program.
type IrcControlChannel struct {
	Enabled       chan bool   // Boolean to turn on or off the IRC client
	ConfigChanged chan bool   // Pass new IRC configuration here to update your client
	IRCError      chan error  // Errors and warnings will be posted here for you to deal with
	Message       chan string // When the regex matches, this channel will be updated
}

// IrcClient is a collection of configs, loggers and other structs defined here.
type IrcClient struct {
	logger         *log.Logger
	config         *config.IRCChannel
	conn           *irc.Connection
	controlChannel *IrcControlChannel
	announceLine   *regexp.Regexp
	state          chan ircState
}

// Init returns an IrcClient
func Init(c *config.IRCChannel, logger *log.Logger) *IrcClient {
	ircConn := irc.IRC(c.Nick, c.Key)
	icc := &IrcControlChannel{}
	icc.IRCError = ircConn.ErrorChan()

	ic := &IrcClient{
		logger:         logger,
		config:         c,
		conn:           ircConn,
		controlChannel: icc,
	}

	// Callbacks for various IRC events.
	ic.conn.AddCallback("001", ic.handleWelcome)
	ic.conn.AddCallback("invite", ic.handleInvite)
	ic.conn.AddCallback("message", ic.handleMsg)
	ic.conn.AddCallback("msg", ic.handleMsg)
	ic.conn.AddCallback("privmsg", ic.handleMsg)
	ic.conn.AddCallback("notice", ic.handleMsg)

	ic.state <- stopped
	return ic
}

// EnableFullIrcLogs will turn on logging events from the IRC channel.
func (ic *IrcClient) EnableFullIrcLogs() {
	ic.conn.AddCallback("*", func(e *irc.Event) {
		ic.logger.Println(e.Message())
	})
}

// DisableFullIRCLogs will turn off logging events from the IRC channel.
func (ic *IrcClient) DisableFullIRCLogs() {
	ic.conn.ClearCallback("*")
}

// Start will do just that. Will connect to the IRC server, register and
// authenticate you nickname and join the channel you want to watch. Errors on
// connection will be returned.
func (ic *IrcClient) Start() error {
	go ic.nickRegistration()
	go ic.trackIRCStatus()
	metricUpdate <- 0
	return ic.connectTracker()
}

func (ic *IrcClient) connectTracker() error {
	s := net.JoinHostPort(ic.config.Server, strconv.Itoa(ic.config.Port))
	ic.logger.Printf("Connection to %s commencing.\n", s)
	if err := ic.conn.Connect(s); err != nil {
		return err
	}
	ic.state <- connected
	ircConnectTimestamp.Set(time.Now().Unix())
	return nil
}

func (ic *IrcClient) disconnectTracker() {
	ic.conn.Disconnect()
	ic.state <- disconnected
	metricUpdate <- time.Now().Unix()
}

func (ic *IrcClient) reconnectTracker() error {
	ic.disconnectTracker()
	ic.nickRegistration()
	return ic.connectTracker()
}

func (ic *IrcClient) nickRegistration() {
	for {
		rs := <-ic.state
		switch {
		case rs == nickReg:
			ic.conn.Privmsgf("NickServ", "identify %s", ic.config.Key)
		case rs == nickAuth:
			go ic.watchChannel()
			return
		}
	}
}

func (ic *IrcClient) sendInvite() error {
	if ic.config.InviteCmd == "" {
		return fmt.Errorf("No invite command.")
	}
	invite, err := ic.generateInvite()
	if err != nil {
		return err
	}
	ic.state <- pendingInvite
	ic.conn.Privmsg(ic.config.ChannelOwner, invite)
	return nil
}

func (ic *IrcClient) generateInvite() (string, error) {
	t, err := template.New("invitecmd").Parse(ic.config.InviteCmd)
	if err != nil {
		return "", err
	}
	ib := bytes.NewBuffer([]byte{})
	if err = t.Execute(ib, ic.config); err != nil {
		return "", err
	}
	return ib.String(), nil
}

func (ic *IrcClient) watchChannel() error {
	if ic.config.WatchChannel == "" {
		return fmt.Errorf("No channel to watch given.")
	}
	ic.conn.Join(ic.config.WatchChannel)
	return nil
}

func (ic *IrcClient) handleInvite(e *irc.Event) {
	if ic.config.WatchChannel == "" {
		ic.controlChannel.IRCError <- fmt.Errorf("Ignoring invite event because no channels are tracked.")
		return
	}
	if strings.HasSuffix(e.Message(), ic.config.WatchChannel) {
		ic.conn.Join(ic.config.WatchChannel)
		ic.state <- watching
		if ic.logger != nil {
			ic.logger.SetPrefix(fmt.Sprintf("%s:", ic.config.WatchChannel))
		}
	}
}

func (ic *IrcClient) handleWelcome(e *irc.Event) {
	n := ic.conn.GetNick()
	if n == ic.config.Nick {
		ic.state <- nickReg
		return
	}
	go ic.conn.Nick(ic.config.Nick)
}

func (ic *IrcClient) handleMsg(e *irc.Event) {
	metricUpdate <- time.Now().Unix()
	ic.controlChannel.Message <- e.Message()
	if ircStateMap[watching] == ircStatus.String() {
		go ic.checkForNickAuth(e.Message())
	}
}

func (ic *IrcClient) checkForNickAuth(m string) {
	switch {
	case strings.Contains(m, "is registered"):
		ic.state <- nickReg
	case strings.Contains(m, "Password accepted"):
		ic.state <- nickAuth
	}
}

func (ic *IrcClient) trackIRCStatus() {
	for {
		select {
		// Turn on and off the IRC service.
		case e := <-ic.controlChannel.Enabled:
			if e && !ic.conn.Connected() {
				ic.controlChannel.IRCError <- ic.Start()
			} else {
				ic.disconnectTracker()
			}
		// Update the IRC configuration.
		case c := <-ic.controlChannel.ConfigChanged:
			if c {
				ic.logger.Println("IRC channel resetting.")
				ic.controlChannel.IRCError <- ic.reconnectTracker()
			}
		// Updates the lastest timestamp metric.
		case ts := <-metricUpdate:
			last, _ := strconv.Atoi(ircUpdateTimestamp.String())
			if ts > int64(last) {
				ircUpdateTimestamp.Set(ts)
			}
		case <-ic.state:
			ic.updateState()
		}
	}
}

func (ic *IrcClient) updateState() {
	ircStatus.Set(ircStateMap[<-ic.state])
}
