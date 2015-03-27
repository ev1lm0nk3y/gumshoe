package gumshoe

import (
	"expvar"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	lastFetch      = expvar.NewInt("last_fetch_timestamp") // timestamp of last successful fetch
	fetchResultMap = expvar.NewMap("fetch_results").Init() // map of fetch return code counters
	downloadQueue  = make(chan int, 10)
)

func init() {
	lastFetch.Set(int64(0))
}

func UpdateResultMap(r string) {
	if fetchResultMap.Get(r) == nil {
		fetchResult := expvar.NewInt(r)
		fetchResult.Set(int64(0))
		fetchResultMap.Set(r, fetchResult)
	}
	fetchResultMap.Add(r, 1)
}

func RetrieveEpisode(link string, tc *TrackerConfig) {
	downloadQueue <- 1
	go func() {
		u, err := url.Parse(link)
		if err != nil {
			log.Println(err)
			<-downloadQueue
			return
		}
		jar, _ := cookiejar.New(nil)
		jar.SetCookies(u, tc.Cookiejar)
		c := &http.Client{Jar: jar}
		sleepDuration := time.Duration(GetRandom(int64(tc.Download.Rate)).Int63())
		time.Sleep(sleepDuration * time.Second)
		log.Printf("Retrieving %s", link)
		resp, err := c.Get(link)
		if err != nil {
			log.Println(err)
			<-downloadQueue
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			<-downloadQueue
			return
		}
		_, dlFile := filepath.Split(u.RequestURI())
		err = ioutil.WriteFile(filepath.Join(tc.Files["base_dir"], tc.Files["torrent_dir"], dlFile), body, os.ModePerm)
		if err != nil {
			log.Println(err)
			<-downloadQueue
			return
		}
		lastFetch.Set(time.Now().Unix())
		UpdateResultMap(strconv.Itoa(resp.StatusCode))
		log.Println("Finished retrieving file.")
		<-downloadQueue
	}()
}
