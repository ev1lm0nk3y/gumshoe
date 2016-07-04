package http

import (
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/ev1lm0nk3y/gumshoe/db"
	"github.com/ev1lm0nk3y/gumshoe/irc"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/oauth2"
	_ "github.com/martini-contrib/secure"
	_ "github.com/martini-contrib/sessions"
	"github.com/martini-contrib/web"
	goauth2 "golang.org/x/oauth2"
)

const (
	SESSION_COOKIE_STORE = "x3Ocy6mHCukBDIPv6meboTKsmyJYCqo4ACk3qbYujZY="
)

var (
	HTTP_HOST = map[string]string{
		"DEV":     "http://gumshoe.evilshatner.com:9119",
		"LAPTOP":  "http://localhost:9119",
		"LOCUTUS": "https://home.evilshatner.com",
		"PROD":    "https://gumshoe.evilshatner.com",
	}
	hs *HttpControlChannel
	tc *config.TrackerConfig
)

// Status is a collection of stats that tell the server if it is healthy or not.
type Status struct {
	IsHealthy             bool
	Uptime                string
	LastSeenWatcherUpdate string
	WatcherStatus         string
	LastEpisodeDownloaded string
}

// HttpControlChannel is a collection of channels to communicate with the calling program.
type HttpControlChannel struct {
	UpdatedCfg   chan bool // Signals when the config was updated via the web
	NumConnected chan int  // Keeps a running tally of the number of open connections.
	HttpRunning  chan bool // Is the http server in a running state. And should it still?
}

func getShows(res http.ResponseWriter) string {
	data, err := db.ListShows()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return err.Error()
	}
	return render(res, data)
}

func getShow(res http.ResponseWriter, params martini.Params) string {
	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err == nil {
		data, err := db.GetShow(id)
		if err == nil {
			return render(res, data)
		}
	}
	return render(res, err)
}

func createShow(res http.ResponseWriter, params martini.Params, show db.Show) string {
	ns := db.NewShow(show.Title, show.Quality, show.Episodal)
	err := ns.AddShow()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return err.Error()
	}
	return render(res, ns)
}

func updateShow(res http.ResponseWriter, params martini.Params, show db.Show) string {
	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err == nil {
		temp := db.NewShow(show.Title, show.Quality, show.Episodal)
		temp.ID = id
		err = temp.UpdateShow()
	}
	return render(res, err)
}

func deleteShow(res http.ResponseWriter, params martini.Params) string {
	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err == nil {
		show := &db.Show{ID: id}
		err = show.DeleteShow()
	}
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	}
	return render(res, err)
}

func getEpisodes(res http.ResponseWriter, params martini.Params) string {
	sid, err := strconv.ParseInt(params["id"], 10, 64)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return err.Error()
	}
	e, err := db.GetEpisodesByShowID(sid)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return err.Error()
	}
	return render(res, e)
}

func getConfig(res http.ResponseWriter, params martini.Params) string {
	if s, ok := params["section"]; ok {
		o, err := tc.GetConfigOption(s)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return err.Error()
		}
		return asJson(res, o)
	}
	o, err := json.Marshal(tc)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return err.Error()
	}
	return asJson(res, o)
}

func updateConfig(res http.ResponseWriter, params martini.Params) string {
	if s, ok := params["config"]; ok {
		err := tc.UpdateGumshoeConfig([]byte(s))
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return err.Error()
		}
		return "Configuration Updated Successfully"
	}
	res.WriteHeader(http.StatusInternalServerError)
	hs.UpdatedCfg <- true
	return "Invalid. Error Code: 10"
}

func getStatus(res http.ResponseWriter, params martini.Params) string {
	/*
	   dateFormat := "Jan 01 2016 @ 12:15pm"
	   statusTmpl, err := template.New("Status").ParseFiles(filepath.Join(tc.Directories["gumshoe_dir"], "www", "templates", "status.html"))
	   if err != nil {
	     res.WriteHeader(http.StatusInternalServerError)
	     return "Internal Error: Code 11"
	   }
	   s := &Status{
	     IsHealthy: true,  //GumshoeHealth(),
	     Uptime: time.Since(time.Unix(int64(strconv.Atoi(expvar.Get("started").String())))),
	     LastSeenWatcherUpdate: time.Unix(int64(strconv.Atoi(expvar.Get("irc_last_update_timestamp").String()))).Format(dateFormat),
	     WatcherStatus: expvar.Get("irc_status"),
	     LastEpisodeDownloaded: time.Unix(int64(strconv.Atoi(expvar.Get("last_fetch_timestamp").String()))).Format(dateFormat),
	   }
	   err = statusTmpl.Execute(res, s)
	   if err != nil {
	     res.WriteHeader(http.StatusInternalServerError)
	     return err
	   }
	*/
	return "OK"
}

func getSettings(res http.ResponseWriter, params martini.Params) string {
	return render(res, tc)
}

func render(res http.ResponseWriter, data interface{}) string {
	thing, err := json.Marshal(data)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return err.Error()
	}
	return asJson(res, thing)
}

func getVarz(res http.ResponseWriter) string {
	const vars = "<b>{{.Key}}:</b> {{.Value.String}}<br>"
	vartmpl := template.Must(template.New("varz").Parse(vars))
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	expvar.Do(func(kv expvar.KeyValue) {
		err := vartmpl.Execute(res, kv)
		if err != nil {
			log.Println(err)
			return
		}
	})
	return ""
}

func asJson(res http.ResponseWriter, data []byte) string {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Access-Control-Allow-Origin", "*")
	return string(data[:])
}

func Verify() {
	return
}

// StartHTTPServer start a HTTP server for configuration and monitoring
func StartHTTPServer(gtc *config.TrackerConfig, l irc.Logger) *HttpControlChannel {
	baseDir := gtc.Directories["gumshoe_dir"]
	tc = gtc
	hs = &HttpControlChannel{}
	m := martini.Classic()

	m.Use(web.ContextWithCookieSecret(SESSION_COOKIE_STORE))
	m.Use(martini.Logger())
	m.Use(oauth2.Google(
		&goauth2.Config{
			ClientID:     os.Getenv("cid"),
			ClientSecret: os.Getenv("cst"),
			Scopes:       []string{"https://www.googeapis.com/auth/drive"},
			RedirectURL:  filepath.Join(HTTP_HOST[os.Getenv("GUMSHOE_ENV")], "/oauth2callback"),
		},
	))

	static := martini.Static(filepath.Join(baseDir, "www"), martini.StaticOptions{Fallback: "/index.html", Exclude: "/api"})
	m.NotFound(static, http.NotFound)

	m.Get("/status", getStatus)
	m.Get("/settings", oauth2.LoginRequired, getSettings)
	m.Get("/vars", getVarz)
	m.Get("/logs", func(res http.ResponseWriter, l irc.Logger) {
		res.WriteHeader(http.StatusOK)
		for _, log := range l() {
			res.Write([]byte(log))
			res.Write([]byte("\n"))
		}
	})

	m.Get("/api/shows", getShows)

	m.Group("/api/show", func(r martini.Router) {
		r.Get("/:id", getShow)
		r.Post("/new", binding.Bind(db.Show{}), createShow)
		r.Post("/update/:id", binding.Bind(db.Show{}), updateShow)
		r.Delete("/delete/:id", deleteShow)
	}, oauth2.LoginRequired)

	m.Group("/api/config", func(r martini.Router) {
		r.Get("/:id", getConfig)
		r.Post("/update", updateConfig)
	}, oauth2.LoginRequired)

	log.Println("Starting up webserver on port %s\n", tc.Operations.HttpPort)
	hostString := fmt.Sprintf(":%s", gtc.Operations.HttpPort)
	go m.RunOnAddr(hostString)
	return hs
}
