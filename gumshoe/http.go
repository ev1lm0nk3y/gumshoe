package gumshoe

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

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
	return render(res, err)
}

func getConfigs() string {
	return "getConfigs"
}

func createConfig() string {
	return "createConfig"
}

func updateConfig() string {
	return "updateConfig"
}

func deleteConfig() string {
	return "deleteConfig"
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

func getStatus() string {
	return "OK"
}

func render(res http.ResponseWriter, data interface{}) string {
	thing, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return asJson(res, thing)
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

	m.Get("/api/shows", getShows)
	m.Group("/api/show", func(r martini.Router) {
		r.Get("/:id", getShow)
		r.Post("/new", binding.Bind(Show{}), createShow)
		r.Post("/update/:id", binding.Bind(Show{}), updateShow)
		r.Delete("/delete/:id", deleteShow)
	})

	m.Group("/api/config", func(r martini.Router) {
		r.Get("/:id", getConfigs)
		r.Post("/new", createConfig)
		r.Put("/update/:id", updateConfig)
		r.Delete("/delete/:id", deleteConfig)
	})

	m.Group("/api/queue", func(r martini.Router) {
		r.Get("/:id", getQueueItems)
		r.Post("/new", createQueueItem)
		r.Delete("/delete/:id", deleteQueueItem)
	})

	log.Println("Starting up webserver...")
	m.RunOnAddr(hostString)
}
