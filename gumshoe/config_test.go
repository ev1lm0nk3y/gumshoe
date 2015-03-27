package gumshoe

import (
  "strings"
  "errors"
  "path/filepath"
  "testing"
  "os"

  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/mock"
)

var (
  pwd, _ = os.Getwd()
  configFile = filepath.Join(pwd, "test_data", "config.json")
  badConfigFile = filepath.Join(pwd, "test_data", "badconfig.json")
  malformedConfigFile = filepath.Join(pwd, "test_data", "malformed.json")
)

var tc_test = &TrackerConfig{
  Files: map[string]string{
    "base_dir": pwd,
  },
  IMDB: IMDBConfig{
    User: "",
    Pass: "",
    Uid: 1,
  },
  IRC: IRCChannel{
    Nick: "test",
    Key: "testkey",
    Server: "test.test.org",
    IRCPort: 2021,
    Debug: true,
  },
  Download: Download{
    Secure: false,
    QueueSize: 1,
  },
  LastModified: int64(0),
}

type MockTrackerConfig struct {
  mock.Mock
  TrackerConfig
}

func (m *MockTrackerConfig) ProcessGumshoeJSON(c string) error {
  args := m.Called(c)
  return args.Error(0)
}

func TestLoadGumshoeConfig_success(t *testing.T) {
  tc := new(MockTrackerConfig)
  tc.On("ProcessGumshoeJSON", configFile).Return(nil)
  if !assert.Nil(t, tc.LoadGumshoeConfig(configFile)) {
    t.Error("Errors seen while processing config file.")
  }
}

func TestLoadGumshoeConfig_failure(t *testing.T) {
  tc := new(MockTrackerConfig)
  tc.On("ProcessGumshoeJSON", badConfigFile).Return(os.PathError{Op: "Open", Path: badConfigFile, Err: errors.New("no such file or directory")})
  tc.On("ProcessGumshoeJSON", "config/gumshoe_config.json").Return(errors.New("file does not exist."))
  // badConfigFile doesn't exist, return os.PathError
  assert.NotNil(t, tc.LoadGumshoeConfig(badConfigFile))
  t.Log(tc.Calls)
  //tc.AssertNumberOfCalls(t, "ProcessGumshoeJSON", 2)
}

func TestProcessGumshoeJSON_success(t *testing.T) {
  tc := NewTrackerConfig()
  if !assert.Nil(t, tc.ProcessGumshoeJSON(configFile)) {
    t.Error("Errors seen while processing config file.")
  }
  if !assert.ObjectsAreEqualValues(tc, tc_test) {
    t.Errorf("Objects don't match:\n\tExpected: %s\n\tActual: %s", tc_test.String(), tc.String())
  }
}

func TestProcessGumshoeJSON_failure(t *testing.T) {
  tc := new(MockTrackerConfig)
  assert.NotNil(t, tc.TrackerConfig.ProcessGumshoeJSON(badConfigFile))
  tc.AssertNumberOfCalls(t, "json.Unmarshal", 0)
}

func TestCreateLocalPath(t *testing.T) {
  tc := NewTrackerConfig()
  tc.Files = map[string]string{"base_dir": pwd}
  clp := tc.CreateLocalPath("config.json", "test_data")
  if !assert.Equal(t, clp, configFile) {
    t.Errorf("Results to not match:\n\tExpected: %s\n\tActual:   %s\n", configFile, clp)
  }
}

func TestSetTrackerCookies(t *testing.T) {
  tc := new(MockTrackerConfig)
  assert.Nil(t, tc.LoadGumshoeConfig(configFile))
  tc.Download = Download{
    Secure: true,
    Cookies: []map[string]string{},
  }
  tc.Download.Cookies = append(tc.Download.Cookies, map[string]string{
    "Name": "user",
    "Value": "tester",
    "Path": "/",
    "Domain": "test.com",
    "Expires": "1427476373",
  })
  tc.Download.Cookies = append(tc.Download.Cookies, map[string]string{
    "Name": "pass",
    "Value": "thistest",
    "Path": "/",
    "Domain": "test.com",
    "Expires": "Never",
  })

  tc.SetTrackerCookies()
  expected := []string{"user=tester; Path=/; Domain=test.com; Expires=Fri, 27 Mar 2015 17:12:53 UTC",
                       "pass=thistest; Path=/; Domain=test.com; Expires=DatePastHereMakesNoDifference"}

  for i := range tc.Cookiejar {
    assert.Equal(t, strings.LastIndex(expected[i], "="), strings.LastIndex(tc.Cookiejar[i].String(), "="))
  }
}
