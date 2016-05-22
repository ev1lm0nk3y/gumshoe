package http

import (
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

  "github.com/ev1lm0nk3y/gumshoe/config"
  "github.com/ev1lm0nk3y/gumshoe/db"
  "github.com/ev1lm0nkey/gumshoe/fetcher"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
)

var tc *config.TrackerConfig

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
    err := tc.UpdateTrackerConfigJSON(s)
    if err != nil {
      res.WriteHeader(http.StatusInternalServerError)
      return err.Error()
    }
    return "Configuration Updated Successfully"
  }
  res.WriteHeader(http.StatusInternalServerError)
  return "Invalid. Error Code: 10"
}

func getStatus(res http.ResponseWriter) string {
  statusTmpl, err := template.New("Status").ParseFile(filepath.Join(tc.Directories["gumshoe_dir"], "www", "templates", "status.html"))
  if err != nil {
    res.WriteHeader(http.StatusInternalServerError)
    return "Internal Error: Code 11"
  }
  s := &Status{
    IsHealthy: GumshoeHealth(),
    Uptime: time.Format(time.Since(time.Unix(expvar.Get("started"))), "Jan 01 2016 @ 12:45pm")
    LastSeenWatcherUpdate: expvar.Get("irc_last_update_timestamp"),
    WatcherStatus: expvar.Get("irc_status"),
    LastEpisodeDownloaded: fetcher.GetLastFetchInfo(),
  }
  err := statusTmpl.Execute(res, s)
  if err != nil {
    res.WriteHeader(http.StatusInternalServerError)
    return err
  }
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
  const vars := "<body>{{range .Vars}}<b>{{.Key}}:</b> {{.Value}}<br>{{end}}</body>"
  vartmpl := template.Must(template.New("varz").Parse(vars))
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	expvar.Do(func(kv expvar.KeyValue) {
    err := vartmpl.Execute(res, kv)
		if err != nil {
      res.WriteHeader(http.StatusInternalServerError)
      return "Failure"
    }
	})
  return ""
}

func asJson(res http.ResponseWriter, data []byte) string {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Access-Control-Allow-Origin", "*")
	return string(data[:])
}

// StartHTTPServer start a HTTP server for configuration and monitoring
func StartHTTPServer(baseDir, port string, gtc *config.TrackerConfig) {
  hostString := fmt.Sprintf(":%s", port)
  tc = gtc
  m := martini.Classic()

	static := martini.Static(filepath.Join(baseDir, "www"), martini.StaticOptions{Fallback: "/index.html", Exclude: "/api"})
	m.NotFound(static, http.NotFound)

	m.Get("/status", getStatus)
	m.Get("/settings", getSettings)
	m.Get("/vars", getVarz)

	m.Get("/api/shows", getShows)
	m.Get("/api/configs", getSettings)

	m.Group("/api/show", func(r martini.Router) {
		r.Get("/:id", getShow)
		r.Post("/new", binding.Bind(db.Show{}), createShow)
		r.Post("/update/:id", binding.Bind(db.Show{}), updateShow)
		r.Delete("/delete/:id", deleteShow)
	})

	m.Group("/api/config", func(r martini.Router) {
		r.Get("/:id", getConfig)
		r.Post("/update", updateConfig)
	})

	log.Println("Starting up webserver...")
	m.RunOnAddr(hostString)
}
