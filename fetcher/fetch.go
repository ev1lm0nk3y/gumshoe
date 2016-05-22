package fetcher

import (
	"expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	fetchResultMap = expvar.NewMap("fetch_results").Init() // map of fetch return code counters
	lastFetch      = expvar.NewInt("last_fetch_timestamp") // timestamp of last successful fetch
  lastFetchEpisode = expvar.NewString("last_fetched_episode")
)

type FileFetch struct {
	HttpClient   *http.Client
	Url          *url.URL
	SaveLocation string
}

func NewFileFetch(link, dest string, cj []*http.Cookie) (ff *FileFetch, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	ff.Url = u
	ff.HttpClient = &http.Client{}
	if ff.Url != nil {
		jar, _ := cookiejar.New(nil)
		jar.SetCookies(ff.Url, cj)
		ff.HttpClient.Jar = jar
	}
	_, dlFile := filepath.Split(u.RequestURI())
	ff.SaveLocation = filepath.Join(dest, string(dlFile[len(dlFile)-1]))
  return ff, nil
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
  episode, err := db.ParseTorrentString(ff.URL)
  if err != nil {
    return err
  }
  show, err := db.GetShow(episode.ShowID)
  if err != nil {
    return err
  }
  lastFetchEpisode.Set(fmt.Sprintf("%s Season %d Episode %d", show.Title, episode.Season, episode.Episode))
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
