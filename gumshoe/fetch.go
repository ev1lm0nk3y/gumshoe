package gumshoe

import (
  "log"
  "expvar"
  "io/ioutil"
  "os"
  "path/filepath"
  "net/http"
  "net/http/cookiejar"
  "net/url"
  "strconv"
  "time"
)

var (
  lastFetch = expvar.NewInt("last_fetch_timestamp")  // timestamp of last successful fetch
  fetchResultMap = expvar.NewMap("fetch_results").Init()  // map of fetch return code counters
  downloadQueue = make(chan int, 10)
  random rand.Rand
)

func init() {
  lastFetch.Set(int64(0))
  random = rand.New(rand.NewSource(int64(15))
}

func UpdateResultMap(r string) {
  if fetchResultMap.Get(r) == nil {
    fetchResult := expvar.NewInt(r)
    fetchResult.Set(int64(0))
    fetchResultMap.Set(r, fetchResult)
  }
  fetchResultMap.Add(r, 1)
}

func RetrieveEpisode(url string, tc *TrackerConfig) {
  downloadQueue<- 1
  go func() {
    u, err := url.Parse(url)
    if err != nil {
      log.Println(err)
      <-downloadQueue
      return
    }
    c := &http.Client{Jar: cookiejar.SetCookie(u, tc.Cookiejar)}
    time.Sleep(time.Duration(random.Float64() * time.Second))
    log.Printf("Retrieving %s", url)
    resp, err := c.Get(url)
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
    err := ioutil.WriteFile(
      filepath.Join(tc.Files["base_dir"],
                    tc.Files["torrent_dir"],
                    dlFile),
      body, os.ModePerm)
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
