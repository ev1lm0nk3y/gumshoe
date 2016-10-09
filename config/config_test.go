package config

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"testing"
)

var (
	tc_test = &TrackerConfig{
		Directories: Directories{
			Main:     "/tmp",
			User:     "/home/foo/.gumshoe",
			Data:     "/home/foo/.gumshoe/data",
			Log:      "/home/foo/.gumshoe/logs",
			Fetch:    "/home/foo/.gumshoe/fetch",
			Download: "/home/foo/.gumshoe/download",
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
)

func TestNew(t *testing.T) {
	type args struct {
		cfg io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *TrackerConfig
		wantErr bool
	}{
		{
			name: "success",
			args: args{cfg: bytes.NewBufferString(`{
	"dir_options": {
		"gumshoe_dir": "foo",
		"user_dir": "bar"
	},
	"download_params": {
		"tracker": "fake",
		"is_secure": false
	},
	"irc_channel": {
		"nick": "fakename",
		"key": "1234567890",
		"server": "www.fake.com"
	},
	"last_modified": 123456789,
	"operations": {
		"enable_web": true,
		"http_port": "1",
		"watch_methods": {
			"irc": true
		}
	}
}
`)},
			want: &TrackerConfig{
				Directories: Directories{
					Main: "foo",
					User: "bar",
				},
				Download: Download{
					Tracker: "fake",
					Secure:  false,
				},
				IRC: IRCChannel{
					Nick:   "fakename",
					Key:    "1234567890",
					Server: "www.fake.com",
				},
				Operations: Operations{
					EnableWeb: true,
					HttpPort:  "1",
					WatchMethods: map[string]bool{
						"irc": true,
					},
				},
				LastModified: int64(123456789),
			},
			wantErr: false,
		},
		{
			name:    "readError",
			args:    args{cfg: bytes.NewBuffer([]byte{})},
			want:    &defaultConfig,
			wantErr: true,
		},
		{
			name:    "invalidJson",
			args:    args{cfg: bytes.NewBufferString("not json")},
			want:    &defaultConfig,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		got, err := New(tt.args.cfg)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. New() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. New() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTrackerConfig_Write(t *testing.T) {
	type fields struct {
		cookieJar    []*http.Cookie
		Directories  Directories
		Download     Download
		IRC          IRCChannel
		LastModified int64
		Operations   Operations
	}
	tests := []struct {
		name    string
		fields  fields
		wantCfg string
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Directories: Directories{
					Main: "foo",
					User: "bar",
				},
				LastModified: int64(0),
				Operations: Operations{
					EnableWeb: true,
					HttpPort:  "1",
				},
			},
			wantCfg: `{
	"dir_options": {
		"gumshoe_dir": "foo",
		"user_dir": "bar"
	},
	"download_params": {},
	"irc_channel": {},
	"last_modified": 0,
	"operations": {
		"enable_web": true,
		"http_port": "1"
	}
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tc := &TrackerConfig{
			cookieJar:    tt.fields.cookieJar,
			Directories:  tt.fields.Directories,
			Download:     tt.fields.Download,
			IRC:          tt.fields.IRC,
			LastModified: tt.fields.LastModified,
			Operations:   tt.fields.Operations,
		}
		cfg := &bytes.Buffer{}
		if err := tc.Write(cfg); (err != nil) != tt.wantErr {
			t.Errorf("%q. TrackerConfig.Write() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if gotCfg := cfg.String(); gotCfg != tt.wantCfg {
			t.Errorf("%q. TrackerConfig.Write() = %v, want %v", tt.name, gotCfg, tt.wantCfg)
		}
	}
}

func TestTrackerConfig_Update(t *testing.T) {
	type fields struct {
		cookieJar    []*http.Cookie
		Directories  Directories
		Download     Download
		IRC          IRCChannel
		LastModified int64
		Operations   Operations
	}
	type args struct {
		update []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantCfg *TrackerConfig
		wantErr bool
	}{
		{
			name:   "success",
			fields: fields{},
			args: args{update: bytes.NewBufferString(`{
				"irc_channel": {
					"nick": "fakenick",
					"key": "fakekey"
				}`).Bytes(),
				wantCfg: &TrackerConfig{
					IRC: IRCChannel{
						Nick: "fakenick",
						Key:  "fakekey",
					},
				},
				wantErr: false,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		tc := &TrackerConfig{
			cookieJar:    tt.fields.cookieJar,
			Directories:  tt.fields.Directories,
			Download:     tt.fields.Download,
			IRC:          tt.fields.IRC,
			LastModified: tt.fields.LastModified,
			Operations:   tt.fields.Operations,
		}
		if err := tc.Update(tt.args.update); (err != nil) != tt.wantErr {
			t.Errorf("%q. TrackerConfig.Update() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
		if !reflect.DeepEqual(tc, wantCfg) {
			t.Errorf("%q, TrackerConfig.Update() error = %v, want %v", tc, wantCfg)
		}
	}
}

func TestTrackerConfig_String(t *testing.T) {
	type fields struct {
		cookieJar    []*http.Cookie
		Directories  Directories
		Download     Download
		IRC          IRCChannel
		LastModified int64
		Operations   Operations
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		tc := &TrackerConfig{
			cookieJar:    tt.fields.cookieJar,
			Directories:  tt.fields.Directories,
			Download:     tt.fields.Download,
			IRC:          tt.fields.IRC,
			LastModified: tt.fields.LastModified,
			Operations:   tt.fields.Operations,
		}
		if got := tc.String(); got != tt.want {
			t.Errorf("%q. TrackerConfig.String() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTrackerConfig_Json(t *testing.T) {
	type fields struct {
		cookieJar    []*http.Cookie
		Directories  Directories
		Download     Download
		IRC          IRCChannel
		LastModified int64
		Operations   Operations
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		tc := &TrackerConfig{
			cookieJar:    tt.fields.cookieJar,
			Directories:  tt.fields.Directories,
			Download:     tt.fields.Download,
			IRC:          tt.fields.IRC,
			LastModified: tt.fields.LastModified,
			Operations:   tt.fields.Operations,
		}
		if got := tc.Json(); got != tt.want {
			t.Errorf("%q. TrackerConfig.Json() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTrackerConfig_postProcess(t *testing.T) {
	type fields struct {
		cookieJar    []*http.Cookie
		Directories  Directories
		Download     Download
		IRC          IRCChannel
		LastModified int64
		Operations   Operations
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		tc := &TrackerConfig{
			cookieJar:    tt.fields.cookieJar,
			Directories:  tt.fields.Directories,
			Download:     tt.fields.Download,
			IRC:          tt.fields.IRC,
			LastModified: tt.fields.LastModified,
			Operations:   tt.fields.Operations,
		}
		if err := tc.postProcess(); (err != nil) != tt.wantErr {
			t.Errorf("%q. TrackerConfig.postProcess() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func Test_readCookieJar(t *testing.T) {
	type args struct {
		cj io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []*http.Cookie
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		got, err := readCookieJar(tt.args.cj)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. readCookieJar() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. readCookieJar() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_writeCookieJar(t *testing.T) {
	type args struct {
		jar []*http.Cookie
	}
	tests := []struct {
		name       string
		args       args
		wantCjFile string
		wantErr    bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		cjFile := &bytes.Buffer{}
		if err := writeCookieJar(cjFile, tt.args.jar); (err != nil) != tt.wantErr {
			t.Errorf("%q. writeCookieJar() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if gotCjFile := cjFile.String(); gotCjFile != tt.wantCjFile {
			t.Errorf("%q. writeCookieJar() = %v, want %v", tt.name, gotCjFile, tt.wantCjFile)
		}
	}
}
