// Package fetcher does the heavy lifting of downloading files from the
// internet to your desired location.
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
)

// FileFetch stores the data about the file to be downloaded.
type FileFetch struct {
	HttpClient   *http.Client
	Url          *url.URL
	SaveLocation string
}

// NewFileFetch will return *FileFetch. Errors may occur if the url is malformed.
func NewFileFetch(link, dest string, cj []*http.Cookie) (*FileFetch, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	ff := &FileFetch{
		HttpClient:   &http.Client{},
		Url:          u,
		SaveLocation: filepath.Join(dest, filepath.FromSlash(u.String())),
	}
	ff.HttpClient.Jar, _ = cookiejar.New(nil)
	ff.HttpClient.Jar.SetCookies(ff.Url, cj)
	return ff, nil
}

// RetrieveEpisode actually makes the http GET call and transfer the data onto
// disk.
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
	updateResultMap(strconv.Itoa(resp.StatusCode))
	return nil
}

// String prints out the details of the file to be fetched.
func (ff *FileFetch) String() string {
	return fmt.Sprintf("URL:\t%s\nDest:\t%s\nTime:\t%s\n",
		ff.Url.String(), ff.SaveLocation, time.Now().String())
}

// updateResultMap takes the http response codes and increments the proper
// expvar counter.
func updateResultMap(r string) {
	if fetchResultMap.Get(r) == nil {
		fr := expvar.NewInt(r)
		fr.Set(int64(1))
		fetchResultMap.Set(r, fr)
		return
	}
	fetchResultMap.Add(r, 1)
}
