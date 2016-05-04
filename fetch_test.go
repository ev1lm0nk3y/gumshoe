package main

import (
	"bytes"
	"expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	urls         [][]byte
	mhc          *MockHttpClient
	mt           *MockTime
	testDataDir  string
	fakeResponse = &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString("fake fetch")),
	}
)

type MockHttpClient struct {
	mock.Mock
	http.Client
}

func (c *MockHttpClient) Get(url string) (res *http.Response, err error) {
	args := c.Called(url)
	return args.Get(0).(*http.Response), args.Error(1)
}

type MockTime struct {
	mock.Mock
	time.Time
}

func (mt *MockTime) Sleep(t time.Duration) {
	return
}

func setUp() {
	tc.LoadGumshoeConfig(configFile)
	testDataDir = filepath.Join(pwd, "test_data")
	testUrls, err := ioutil.ReadFile(filepath.Join(testDataDir, "test_urls"))
	if err != nil {
		fmt.Print(err)
		os.Exit(5)
	}
	urls = bytes.Split(bytes.TrimSpace(testUrls), []byte("\n"))

	mhc = &MockHttpClient{}
	mhc.On("Get", mock.AnythingOfType("string")).Return(
		&http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString("fake fetch"))},
		nil)

	mt = &MockTime{}
	mt.On("Sleep", mock.AnythingOfType("time.Duration"))
}

func tearDown() {
	tc = NewTrackerConfig()
	testDataDir = ""
	urls = [][]byte{}
	mhc = &MockHttpClient{}
	mt = &MockTime{}
}

func TestSetClientCookie(t *testing.T) {
	tUrl, _ := url.Parse("http://www.evilshatner.com/download.php/344551/conan.2015.03.26.will.ferrell.hdtv.x264-daview.mp4.torrent")
	ff := &FileFetch{
		HttpClient: &http.Client{},
		Url:        tUrl}
	tj, _ := cookiejar.New(nil)
	tj.SetCookies(ff.Url, cj)
	err := ff.SetClientCookie()
	assert.Nil(t, err)
	assert.Equal(t, ff.HttpClient.Jar, tj)
}

func TestRetrieveEpisode(t *testing.T) {
	setUp()
	start := time.Now()
	var num_episodes int
	for _, u := range urls {
		err := AddEpisodeToQueue(string(u))
		if err != nil {
			t.Fatal(err)
		}
		num_episodes = num_episodes + 1
	}
	for episodeQueue.Len() > 0 {
		f := episodeQueue.PopFront().(*FileFetch)
		err := f.RetrieveEpisode()
		assert.NoError(t, err)
		assert.NotEqual(t, lastFetch.String(), string(start.Unix()))
		assert.NotNil(t, fetchResultMap.Get("200"))
	}
	fetchResultMap.Do(func(kv expvar.KeyValue) {
		fmt.Printf("%s: %s\n", kv.Key, kv.Value.String())
	})
	contents, err := ioutil.ReadDir(testDataDir)
	if err != nil {
		t.Fatal(err)
	}
	torrents := []string{}
	for _, f := range contents {
		if f.IsDir() || !strings.Contains(f.Name(), ".torrent") {
			continue
		}
		torrents = append(torrents, f.Name())
	}
	assert.Equal(t, len(urls), len(torrents))

	// Make sure the metrics are being updated
	lft, _ := strconv.Atoi(expvar.Get("last_fetch_timestamp").String())
	assert.True(t, int64(lft) >= start.Unix(), "Timestamp not updated.\n\tStarted: %d\n\tTimestamp: %d", start.Unix(), lft)
	st := strconv.Itoa(len(torrents))
	assert.Equal(t, fmt.Sprintf("{\"200\": %s}", st), expvar.Get("fetch_results").String())
	tearDown()
}

func TestUpdateResultMap(t *testing.T) {
	fetchResultMap = expvar.NewMap("test_fetch_results").Init()
	UpdateResultMap("ryan")
	UpdateResultMap("daniel")
	UpdateResultMap("ryan")
	UpdateResultMap("monkeys")
	fetchResultMap.Do(func(kv expvar.KeyValue) {
		if kv.Key == "ryan" {
			assert.Equal(t, kv.Value.String(), "2")
		} else if kv.Key == "daniel" {
			assert.Equal(t, kv.Value.String(), "1")
		} else if kv.Key == "monkeys" {
			assert.Equal(t, kv.Value.String(), "1")
		} else {
			t.Fatal("A value in the result map is messing things up.")
		}
	})
}

func TestAddEpisodeToQueue(t *testing.T) {
	setUp()
	for _, u := range urls {
		err := AddEpisodeToQueue(string(u))
		assert.Nil(t, err)
	}
	assert.Equal(t, len(urls), episodeQueue.Len())
	for episodeQueue.Len() > 0 {
		e := episodeQueue.PopFront()
		assert.NotNil(t, e)
	}
	tearDown()
}
