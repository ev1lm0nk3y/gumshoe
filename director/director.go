package director

import (
	"expvar"
	"os"
	"time"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/uber-go/zap"
)

// Status can be used by all services to define what state they are in.
type Status int

const (
	NotRunning   Status = iota // The service is not enabled, or has not started
	Starting                   // The service is starting, but has not finished
	Running                    // The service is running.
	ErrorRunning               // The service is in an error state but still running.
	ErrorStopped               // The service stopped when it went into an error state.
)

var (
	_dStatus       = expvar.NewMap("director_status").Init()
	_dStartms      = expvar.NewInt("director-start-ms")
	_dLastUpdate   = expvar.NewInt("director-last-update")
	_configUpdated = make(chan bool)
)

type Director struct {
	Config                          *config.TrackerConfig
	Db, Irc, Http                   Control
	UpdateAll                       chan bool
	dbEnable, IrcEnable, HttpEnable chan bool
	Signal                          chan os.Signal
}

type Control struct {
	Config  interface{}
	Enabled _func
	Error   chan []zap.Field
	Status  chan Status
	Msg     chan []zap.Field
	Updated _func
	Logger  zap.Logger
}

type _func func(...interface{}) error

func New(tc *config.TrackerConfig, db, irc, http Control, sig chan os.Signal) *Director {
	_dStartms.Set(time.Now().Unix())
	return &Director{
		Config: tc,
		Db:     db,
		Irc:    irc,
		Http:   http,
		Signal: sig,
	}
}

func NewController(tc interface{}, e, u _func) (*Control, error) {
	cc := &Control{
		Config:  tc,
		Enabled: e,
		Error:   make(chan []zap.Field, 10),
		Msg:     make(chan []zap.Field, 10),
		Updated: u,
		Status:  make(chan Status),
	}
	if ilog, ok := cc.Config.InfoLogPath; ok {
		cc.Logger.NewInfoLogger(cc.Config.InfoLogPath, zap.InfoLevel)
		if elog, ok := cc.Config.ErrLogPath; ok {
			cc.Logger.AddErrorLogger(elog, zap.ErrorLevel)
		}
	}
	return cc, nil
}

func (d *Director) Run() {
	d.UpdateAll = make(chan bool)
	d.dbEnable = make(chan bool)
	d.IrcEnable = make(chan bool)
	d.HttpEnable = make(chan bool)

	go d.Db.Manage(d.UpdateAll, d.dbEnable)
	go d.Irc.Manage(d.UpdateAll, d.IrcEnable)
	go d.Http.Manage(d.UpdateAll, d.HttpEnable)
}

func (c *Control) Manage(up, run <-chan bool) {
	if c == nil {
		return
	}
	for {
		select {
		// Msg entries go into the info log
		case msg := <-c.Msg:
			c.Logger.Info("", msg...)
		// Error entries go into the error log
		case elog := <-c.Error:
			c.Logger.Error("", elog...)
		// Trigger an update procedure.
		case <-up:
			err := c.Updated()
			if err != nil {
				f := zap.String("msg", err.Error())
				c.Error <- []zap.Field{f}
			}
		// Trigger either a disable or enabling of the service.
		case <-run:
			err := c.Enabled()
			if err != nil {
				f := zap.String("msg", err.Error())
				c.Error <- []zap.Field{f}
			}
		}
	}
}

func NewInfoLogger(f string, l zap.Level) (zap.Logger, error) {
	f = append(f, "-%s-%d.INFO.log", time.Now().Format("%%Y%%m%%d"), os.Getpid())
	fw, err := os.Create(f)
	if err != nil {
		return nil, err
	}
	l := zap.NewJSON(level == l, w == zap.AddSync(fw), alwaysEpoch == false)
}

func AddErrorLogger(z *zap.Logger, f string, l zap.Level) (zap.Logger, error) {
	f = append(f, "-%s-%d.ERROR.log", time.Now().Format("%%Y%%n%%d"), os.Getpid())
	fw, err := os.Create(f)
	if err != nil {
		return nil, err
	}
	return z.With(level == l, errW == zap.AddSync(fw), alwaysEpoch == false), nil
}
