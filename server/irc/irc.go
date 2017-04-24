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
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
type Client struct {
	Enabled  chan bool              // Boolean to turn on or off the IRC client
	Config   chan config.IRCChannel // Pass new IRC configuration here to update the irc connection
	IRCError chan error             // Errors and warnings will be posted here for you to deal with
	Log      chan string            // Log will be the IRCs logging channel
	Message  chan string            // When the regex matches, this channel will be updated

	config       config.IRCChannel
	conn         *irc.Connection
	announceLine *regexp.Regexp
	state        chan ircState

	mutex sync.RWMutex
}

// Init returns a Client object.
func Init() *Client {
	c := &Client{}

	c.state <- stopped
	metricUpdate <- 0

	go c.controller()
	return c
}

func (c *Client) controller() {
	for {
		select {
		// Turn on and off the IRC service.
		case e := <-c.Enabled:
			if e && !c.conn.Connected() {
				c.IRCError <- c.connectTracker()
			} else {
				c.disconnectTracker()
			}
		// Update the IRC configuration.
		case cfg := <-c.Config:
			c.mutex.Lock()
			c.config = cfg
			c.mutex.Unlock()
			c.Log <- fmt.Sprintf("IRC channel resetting")
			c.IRCError <- c.reconnectTracker()
		// Updates the lastest timestamp metric.
		case ts := <-metricUpdate:
			last, _ := strconv.Atoi(ircUpdateTimestamp.String())
			if ts > int64(last) {
				ircUpdateTimestamp.Set(ts)
			}
		// Just makes sure that the IRC status stay up to date
		case <-c.state:
			c.updateState()
		}
	}
}

func (c *Client) createConn(s string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.conn = irc.IRC(c.config.Nick, c.config.Key)
	c.IRCError = c.conn.ErrorChan()

	// Callbacks for various IRC events.
	c.conn.AddCallback("001", c.handleWelcome)
	c.conn.AddCallback("invite", c.handleInvite)
	c.conn.AddCallback("message", c.handleMsg)
	c.conn.AddCallback("msg", c.handleMsg)
	c.conn.AddCallback("privmsg", c.handleMsg)
	c.conn.AddCallback("notice", c.handleMsg)

	if err := c.conn.Connect(s); err != nil {
		c.IRCError <- fmt.Errorf("IRC connect error: %v", err)
		c.state <- disconnected
	}
	ircConnectTimestamp.Set(time.Now().Unix())
	c.state <- connected
}

func (c *Client) connectTracker() error {
	c.mutex.RLock()
	if c.config.Server == "" {
		return fmt.Errorf("IRC Connect Error: No configuration found")
	}
	s := net.JoinHostPort(c.config.Server, strconv.Itoa(c.config.Port))
	c.Log <- fmt.Sprintf("Connection to %s commencing.\n", s)
	c.mutex.RUnlock()

	// Have to unlock after the initial read to ensure createConn can acquire the
	// write lock.
	go c.createConn(s)
	go c.nickRegistration()
	return nil
}

func (c *Client) disconnectTracker() {
	c.conn.Disconnect()
	c.state <- disconnected
	metricUpdate <- time.Now().Unix()
}

func (c *Client) reconnectTracker() error {
	c.disconnectTracker()
	return c.connectTracker()
}

func (c *Client) nickRegistration() {
	for {
		rs := <-c.state
		switch {
		case rs == nickReg:
			c.mutex.RLock()
			defer c.mutex.RUnlock()
			c.conn.Privmsgf("NickServ", "identify %s", c.config.Key)
		case rs == nickAuth:
			go c.watchChannel()
			return
		}
	}
}

func (c *Client) sendInvite() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.config.InviteCmd == "" {
		return fmt.Errorf("IRC Invite Error: No invite command.")
	}
	invite, err := c.generateInvite()
	if err != nil {
		return fmt.Errorf("IRC Invite Error: %v", err)
	}
	c.state <- pendingInvite
	c.conn.Privmsg(c.config.ChannelOwner, invite)
	return nil
}

func (c *Client) generateInvite() (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	t, err := template.New("invitecmd").Parse(c.config.InviteCmd)
	if err != nil {
		return "", fmt.Errorf("IRC Invite Error: %v", err)
	}
	ib := bytes.NewBuffer([]byte{})
	if err = t.Execute(ib, c.config); err != nil {
		return "", fmt.Errorf("IRC Invite Error: %v", err)
	}
	return ib.String(), nil
}

func (c *Client) handleInvite(e *irc.Event) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.config.WatchChannel == "" {
		c.IRCError <- fmt.Errorf("Ignoring invite event because no channels are tracked")
		return
	}
	if strings.HasSuffix(e.Message(), c.config.WatchChannel) {
		c.conn.Join(c.config.WatchChannel)
		c.state <- watching
	}
}

func (c *Client) handleWelcome(e *irc.Event) {
	n := c.conn.GetNick()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if n == c.config.Nick {
		c.state <- nickReg
		return
	}
	go c.conn.Nick(c.config.Nick)
}

func (c *Client) watchChannel() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.config.WatchChannel == "" {
		return fmt.Errorf("IRC Watch Error: No channel to watch")
	}
	c.conn.Join(c.config.WatchChannel)
	return nil
}

func (c *Client) handleMsg(e *irc.Event) {
	metricUpdate <- time.Now().Unix()
	c.Message <- e.Message()
	if ircStateMap[watching] == ircStatus.String() {
		go c.checkForNickAuth(e.Message())
	}
}

func (c *Client) checkForNickAuth(m string) {
	switch {
	case strings.Contains(m, "is registered"):
		c.state <- nickReg
	case strings.Contains(m, "Password accepted"):
		c.state <- nickAuth
	}
}

func (c *Client) updateState() {
	ircStatus.Set(ircStateMap[<-c.state])
}
