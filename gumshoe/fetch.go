package gumshoe

import (
	"errors"
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

type FileFetch struct {
	HttpClient   *http.Client
	Url          *url.URL
	SaveLocation string
}

func NewFileFetch(link string) (ff *FileFetch, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	ff.Url = u
	ff.HttpClient = &http.Client{}
	err = ff.setClientCookie()
  if err != nil {
		return nil, err
	}
	_, dlFile := filepath.Split(u.RequestURI())
	ff.SaveLocation = filepath.Join(tc.Directories["user_dir"], tc.Directories["torrent_dir"], string(dlFile[len(dlFile)-1]))
  return ff, nil
}

func (ff *FileFetch) setClientCookie() error {
	if ff.Url != nil {
		jar, _ := cookiejar.New(nil)
		jar.SetCookies(ff.Url, cj)
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
