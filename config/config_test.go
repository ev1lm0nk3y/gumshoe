package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	//  "github.com/ev1lm0nk3y/gumshoe/gumshoe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	pwd, _              = os.Getwd()
	configFile          = filepath.Join(pwd, "test_data", "config.json")
	badConfigFile       = filepath.Join(pwd, "test_data", "badconfig.json")
	malformedConfigFile = filepath.Join(pwd, "test_data", "malformed.json")
)

var tc_test = &TrackerConfig{
	Directories: map[string]string{
		"base_dir":    "",
		"data_dir":    "test_data",
		"log_dir":     "test_data",
		"user_dir":    pwd,
		"torrent_dir": filepath.Join(pwd, "test_data"),
	},
	IRC: IRCChannel{
		ChannelOwner:   "BitMeTV",
		Nick:           "test",
		Key:            "testkey",
		Server:         "localhost",
		Port:           6626,
		EnableLog:      true,
		InviteCmd:      "!invite %nick% %key%",
		WatchChannel:   "#announce",
		AnnounceRegexp: "BitMeTV-IRC2RSS%3A%20(%3FP%3Ctitle%3E.*%3F)%20%3A%20(%3FP%3Curl%3E.*)",
		EpisodeRegexp:  "%28%5C%5CS%2B%29%5C%5C.%28%3Fi%3As%28%5C%5Cd%7B2%7D%29e%28%5C%5Cd%7B2%7D%29%7C%28%5C%5Cd%7B4%7D.%5C%5Cd%7B2%7D.%5C%5Cd%7B2%7D%29%7C%28%5C%5Cd%29x%3F%28%5C%5Cd%7B2%7D%29%29%5C%5C.%28.%2B%29%3F%5C%5C.%3F%28%3Fi%3A720p%7C1080p%29%3F%5C%5C.%3F%28%3Fi%3Ahdtv%7Cweb.%2B%29%5C%5C..%2B%5C%5C.torrent",
	},
	Download: Download{
		Tracker:    "localhost",
		Rate:       20,
		Secure:     false,
		QueueSize:  1,
		MaxRetries: 3,
	},
	Operations: Operations{
		EnableLog: true,
		EnableWeb: true,
		HttpPort:  "8080",
		WatchMethods: map[string]bool{
			"irc": true,
			"rss": false,
		},
	},
	LastModified: int64(0),
}

type MockTrackerConfig struct {
	mock.Mock
	mtc *TrackerConfig
}

func (m *MockTrackerConfig) ProcessGumshoeCfgFile(c string) error {
	args := m.Called(c)
	return args.Error(0)
}

/*func (m *MockTrackerConfig) LoadGumshoeConfig(c string) error {
  args := m.Called(c)
  rtc := NewTrackerConfig()
  err := rtc.LoadGumshoeConfig(args.String(0))
  return err
}*/

func TestLoadGumshoeConfig_success(t *testing.T) {
	mtc := NewTrackerConfig()
	tErr := mtc.LoadGumshoeConfig(configFile)
	assert.Nil(t, tErr)
}

func TestLoadGumshoeCfgFile_failure(t *testing.T) {
	mtc := NewTrackerConfig()
	assert.NotNil(t, mtc.LoadGumshoeConfig(badConfigFile))
}

func TestProcessGumshoeCfgFile_success(t *testing.T) {
	tc := NewTrackerConfig()
	if !assert.Nil(t, tc.ProcessGumshoeCfgFile(configFile)) {
		t.Error("Errors seen while processing config file.")
	}
	if !assert.ObjectsAreEqualValues(tc, tc_test) {
		t.Errorf("Objects don't match:\n\tExpected: %s\n\tActual: %s", tc_test.String(), tc.String())
	}
}

func TestProcessGumshoeCfgFile_failure(t *testing.T) {
	mtc := new(MockTrackerConfig)
	mtc.mtc = NewTrackerConfig()
	assert.NotNil(t, mtc.mtc.ProcessGumshoeCfgFile(badConfigFile))
	mtc.AssertNumberOfCalls(t, "json.Unmarshal", 0)
}

func TestProcessGumshoeCfgJson_success(t *testing.T) {
	o, _ := json.MarshalIndent(tc_test, "", "\t")
	tc := NewTrackerConfig()
	err := tc.ProcessGumshoeCfgJson(o)
	if !assert.Nil(t, err) {
		t.Error(err)
	}
	if !assert.Equal(t, string(o), tc.String()) {
		t.Errorf("unmatched strings:\nExpected:%s\nActual:%s", string(o), tc.String())
	}
}

func TestProcessGumshoeCfgJson_failure(t *testing.T) {
	tc := NewTrackerConfig()
	tJson := []byte("{ 'unknown': { 'foo': 'bar', 'baz': false }}")
	err := tc.ProcessGumshoeCfgJson(tJson)
	if !assert.NotNil(t, err) {
		t.Error(tc.String())
	}
}

func TestCreateLocalPath(t *testing.T) {
	tc_test.Directories["user_dir"] = pwd
	tc_test.Directories["data_dir"] = "test_data"
	clp := CreateLocalPath(tc_test, "config.json")
	if !assert.Equal(t, clp, configFile) {
		t.Errorf("Results to not match:\n\tExpected: %s\n\tActual:   %s\n", configFile, clp)
	}
}

func TestUpdateGumshoeConfig(t *testing.T) {
	err := tc_test.UpdateGumshoeConfig([]byte("{\"irc_channel\": {\"owner\": \"test_owner\", \"key\": \"testkey\"}}"))
	if !assert.Nil(t, err) {
		t.Error(err.Error())
	}
	assert.Equal(t, tc_test.IRC.ChannelOwner, "test_owner")
	assert.Equal(t, tc_test.IRC.Key, "testkey")
}

func TestWriteGumshoeConfig(t *testing.T) {
	err := tc_test.WriteGumshoeConfig(".config_write_test")
	if !assert.Nil(t, err) {
		t.Error(err.Error())
	}
	os.Remove(filepath.Join(pwd, "test_data", ".config_write_test"))
}

func TestSetTrackerCookies(t *testing.T) {
	mtc := new(MockTrackerConfig)
	mtc.mtc = NewTrackerConfig()
	assert.Nil(t, mtc.mtc.LoadGumshoeConfig(configFile))
	mtc.mtc.Download.Secure = true
	mtc.mtc.Directories["data_dir"] = "test_data"
	mtc.mtc.Directories["user_dir"] = pwd

	assert.Nil(t, mtc.mtc.SetTrackerCookies())

	expected := []string{"user=tester; Path=/; Domain=test.com; Expires=Fri, 27 Mar 2015 17:12:53 UTC",
		"pass=thistest; Path=/; Domain=test.com; Expires=DatePastHereMakesNoDifference"}

	for i, cookie := range GetTrackerCookies() {
		assert.Equal(t, strings.LastIndex(expected[i], "="), strings.LastIndex(cookie.String(), "="))
	}
}

func TestGetConfigOption(t *testing.T) {
	b, err := tc_test.GetConfigOption("dir_options")
	assert.Nil(t, err)
	x, err := json.Marshal(tc_test.Directories)
	assert.Nil(t, err)
	assert.Equal(t, b, x)
	b, err = tc_test.GetConfigOption("operations")
	assert.Nil(t, err)
	x, err = json.Marshal(tc_test.Operations)
	assert.Nil(t, err)
	assert.Equal(t, b, x)
	b, err = tc_test.GetConfigOption("download_params")
	assert.Nil(t, err)
	x, err = json.Marshal(tc_test.Download)
	assert.Nil(t, err)
	assert.Equal(t, b, x)
	b, err = tc_test.GetConfigOption("irc_channel")
	assert.Nil(t, err)
	x, err = json.Marshal(tc_test.IRC)
	assert.Nil(t, err)
	assert.Equal(t, b, x)
	_, err = tc_test.GetConfigOption("xxx")
	assert.NotNil(t, err)
}
