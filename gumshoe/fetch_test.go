package gumshoe

import (
  "bytes"
  "expvar"
  "fmt"
  "io/ioutil"
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "testing"
  "time"

  "github.com/ev1lm0nk3y/gumshoe/test"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/mock"
)

var (
  testUrls []byte
  tc = NewTrackerConfig()
  fakeResponse = &http.Response{
    Status: "200 OK",
    StatusCode: 200,
    Body: ioutil.NopCloser(bytes.NewBufferString("fake fetch")),
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

var MockSleep = func(t time.Duration) {
  fmt.Printf("This is where you would sleep for %s", t.String())
  if t.Seconds() < 0 || t.Seconds() > float64(tc.Download.Rate) {
    fmt.Errorf("Random Time is out of bounds.")
  }
}

func init() {
}

func TestRetrieveEpisode(t *testing.T) {
  tc.LoadGumshoeConfig(configFile)
  testUrls, err := ioutil.ReadFile(filepath.Join(pwd, "test_data", "test_urls"))
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  urls := bytes.Split(testUrls, []byte("\n"))

  mc := new(MockHttpClient)
  mc.On("Get", mock.AnythingOfType("string")).Return(fakeResponse, nil)

  start := time.Now()
  defer test.Patch(&MockSleep, func(x time.Duration) {
    for i := range urls {
      url := string(urls[i][0:])
      fmt.Println(url)
      RetrieveEpisode(url, tc)
    }
    contents, err := ioutil.ReadDir(filepath.Join(pwd, "test_data"))
    if err != nil {
      t.Fatal(err)
    }
    torrents := []string{}
    for f := range contents {
      if contents[f].IsDir() || !strings.Contains(contents[f].Name(), ".torrent") {
        continue
      }
      torrents = append(torrents, contents[f].Name())
    }
    assert.Equal(t, len(urls), len(torrents))
    lft, _ := strconv.Atoi(expvar.Get("last_fetch_timestamp").String())
    assert.True(t, int64(lft) > start.Unix(), "Timestamp not updated.")
    st := strconv.Itoa(len(torrents))
    assert.Equal(t, st, expvar.Get("fetch_results").String())
  }).Restore()
}

