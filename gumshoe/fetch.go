package gumshoe

import (
  "errors"
	"expvar"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
  "path/filepath"
	"strconv"
	"time"

	"github.com/yasushi-saito/fifo_queue"
)

var (
	lastFetch      = expvar.NewInt("last_fetch_timestamp") // timestamp of last successful fetch
	fetchResultMap = expvar.NewMap("fetch_results").Init() // map of fetch return code counters
	queueDepth     int
	episodeQueue   *fifo_queue.Queue
	downloader_on  = make(chan bool)
  process_queue  = make(chan bool)
)

func init() {
	lastFetch.Set(int64(0))
	episodeQueue = fifo_queue.NewQueue()
}

type FileFetch struct {
	HttpClient   *http.Client
	Url          *url.URL
	SaveLocation string
}

func (ff *FileFetch) SetClientCookie() error {
	if ff.Url != nil {
		jar, _ := cookiejar.New(nil)
		jar.SetCookies(ff.Url, tc.Cookiejar)
		ff.HttpClient.Jar = jar
		return nil
	}
	return errors.New("Fetch URL not set.")
}

func (ff *FileFetch) RetrieveEpisode() error {
	resp, err := ff.HttpClient.Get(ff.Url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(ff.SaveLocation, body, os.ModePerm)
	if err != nil {
		return err
	}
	lastFetch.Set(time.Now().Unix())
	UpdateResultMap(strconv.Itoa(resp.StatusCode))
  return nil
}

func (ff *FileFetch) Print() string {
	return fmt.Sprintf("URL:      %s\nLocation: %s\nTime:     %s\n",
		ff.Url.String(), ff.SaveLocation, time.Now().String())
}

func UpdateResultMap(r string) {
	if fetchResultMap.Get(r) == nil {
		fr := expvar.NewInt(r)
		fr.Set(int64(1))
		fetchResultMap.Set(r, fr)
    return
	}
	fetchResultMap.Add(r, 1)
}

func AddEpisodeToQueue(link string) error {
	ff := &FileFetch{}
	u, err := url.Parse(link)
	if err != nil {
		return err
	}
	ff.Url = u
	ff.HttpClient = &http.Client{}
	if err = ff.SetClientCookie(); err != nil {
		return err
	}
	_, dlFile := filepath.Split(u.RequestURI())
	s := filepath.Join(tc.Files["base_dir"], tc.Files["torrent_dir"], dlFile)
	ff.SaveLocation = s
	episodeQueue.PushBack(ff)
  return nil
}

func StartDownloader() {
	go func() {
		for {
      process_queue<- (episodeQueue.Len() > 0)
      select {
      case on := <-downloader_on:
				if episodeQueue.Len() > 0 && !on {
					log.Println("Waiting for %d items to finish downloading.", episodeQueue.Len())
					time.Sleep(time.Second * time.Duration(tc.Download.Rate))
          continue
				}
				return
      case do := <-process_queue:
				queueDepth = episodeQueue.Len()
        if do {
          f := episodeQueue.PopFront().(*FileFetch)
				  time.Sleep(time.Duration(GetRandom(int64(tc.Download.Rate)).Int63()))
					err := f.RetrieveEpisode()
					if err != nil {
						log.Println("[%s] Download Error: %s", time.Now().Format("2015-05-30 08:23:45"), err)
						UpdateResultMap("download_errors")
					}
					lastFetch.Set(time.Now().Unix())
        }
			default:
				time.Sleep(time.Minute * 5)
        process_queue<- false
			}
		}
	}()
	downloader_on <- true
  process_queue <- false
}

func StopDownloader() {
	downloader_on <- false
}

func RestartDownloader() {
	downloader_on <- false
	StartDownloader()
}
