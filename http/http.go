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

  "github.com/ev1m0nk3y/gumshoe/db"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
)

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

func createShow(res http.ResponseWriter, params martini.Params, show Show) string {
	ns := db.NewShow(show.Title, show.Quality, show.Episodal)
	err := ns.AddShow()
  if err != nil {
    res.WriteHeader(http.StatusInternalServerError)
    return err.Error()
  }
	return render(res, ns)
}

func updateShow(res http.ResponseWriter, params martini.Params, show Show) string {
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
	//  res.WriteHeader(http.StatusInternalServerError)
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
		res.WriteHeader(http.StatusInternalServerError)
		return err.Error()
	}
	return asJson(res, thing)
}

func getVarz(res http.ResponseWriter) string {
  output := []string{}
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	output = append(output, fmt.Sprintf("{", "\n"))
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			output = append(output, fmt.Sprintf(",\n"))
			first = false
		}
		output = append(output, fmt.Sprintf("%q: %s\n", kv.Key, kv.Value))
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
func StartHTTPServer(baseDir, port string) {
  hostString := fmt.Sprintf(":%s", port)
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
