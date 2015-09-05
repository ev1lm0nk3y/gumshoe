package gumshoe

import (
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
)

func getShows(res http.ResponseWriter) string {
	data, err := ListShows()
	if err == nil {
		return render(res, data)
	}
	return render(res, err)
}

func getShow(res http.ResponseWriter, params martini.Params) string {
	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err == nil {
		data, err := GetShow(id)
		if err == nil {
			return render(res, data)
		}
	}
	return render(res, err)
}

func createShow(res http.ResponseWriter, params martini.Params, show Show) string {
	err := AddShow(show.Title, show.Quality, show.Episodal)
	return render(res, err)
}

func updateShow(res http.ResponseWriter, params martini.Params, show Show) string {
	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err == nil {
		temp := newShow(show.Title, show.Quality, show.Episodal)
		temp.ID = id
		err = UpdateShow(*temp)
	}
	return render(res, err)
}

func deleteShow(res http.ResponseWriter, params martini.Params) string {
	id, err := strconv.ParseInt(params["id"], 10, 64)
	if err == nil {
		show := Show{ID: id}
		err = DeleteShow(show)
	}
  if err != nil {
    res.WriteHeader(500)
  }
	return render(res, err)
}

func getEpisodes(res http.ResponseWriter, params martini.Params) string {
  sid, err := strconv.ParseInt(params["id"], 10, 64)
  if err != nil {
    res.WriteHeader(500)
    return render(res, err)
  }
  e, err := GetEpisodesByShowID(sid)
  if err != nil {
    res.WriteHeader(500)
    return render(res, err)
  }
  return render(res, e)
}

func getConfig(res http.ResponseWriter, params martini.Params) string {
  if s, ok := params["section"]; ok {
    o, err := tc.GetConfigOption(s)
    if err != nil {
      res.WriteHeader(500)
      err.Error()
    }
    return asJson(res, o)
  }
  o, err := json.Marshal(tc)
  if err != nil {
    res.WriteHeader(500)
    return "Invalid Config"
  }
  return asJson(res, o)
}

func updateConfig() string {
	return "updateConfig"
}

func getQueueItems() string {
	return "getQueueItems"
}

func createQueueItem() string {
	return "createQueueItem"
}

func deleteQueueItem() string {
	return "deleteQueueItem"
}

func getStatus(res http.ResponseWriter) string {
  //_, err := torrentClient.GetTorrents()
  //if err != nil {
  //  res.WriteHeader(500)
  //  return err.Error()
  //}
  return "OK"
}

func getSettings(res http.ResponseWriter, params martini.Params) string {
  return render(res, tc)
}

func render(res http.ResponseWriter, data interface{}) string {
	thing, err := json.Marshal(data)
	if err != nil {
    res.WriteHeader(500)
    return err.Error()
	}
	return asJson(res, thing)
}

func getVarz(res http.ResponseWriter) string {
	var output = []string{}
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	output = append(output, fmt.Sprintf("{", "\n"))
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			output = append(output, fmt.Sprintf(",\n"))
			first = false
		}
		output = append(output, fmt.Sprintf("%q: %s", kv.Key, kv.Value))
	})
	output = append(output, "\n}\n")
	return strings.Join(output, "")
}

func asJson(res http.ResponseWriter, data []byte) string {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Access-Control-Allow-Origin", "*")
	return string(data[:])
}

// StartHTTPServer start a HTTP server for configuration and monitoring
func StartHTTPServer(baseDir string, port string) {
	var hostString = fmt.Sprintf(":%s", port)
	var m = martini.Classic()

	static := martini.Static(filepath.Join(baseDir, "www"), martini.StaticOptions{Fallback: "/index.html", Exclude: "/api"})
	m.NotFound(static, http.NotFound)

	m.Get("/status", getStatus)
  m.Get("/settings", getSettings)
	m.Get("/vars", getVarz)

	m.Get("/api/shows", getShows)
  m.Get("/api/configs", getSettings)

	m.Group("/api/show", func(r martini.Router) {
		r.Get("/:id", getShow)
		r.Post("/new", binding.Bind(Show{}), createShow)
		r.Post("/update/:id", binding.Bind(Show{}), updateShow)
		r.Delete("/delete/:id", deleteShow)
	})

  m.Group("/api/config", func(r martini.Router) {
    r.Get("/:id", getConfig)
    r.Post("/update", updateConfig)
  })

	m.Group("/api/queue", func(r martini.Router) {
		r.Get("/:id", getQueueItems)
		r.Post("/new", createQueueItem)
		r.Delete("/delete/:id", deleteQueueItem)
	})

	log.Println("Starting up webserver...")
	m.RunOnAddr(hostString)
}
