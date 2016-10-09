package main

import (
	"log"
	"reflect"
	"testing"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/ev1lm0nk3y/gumshoe/fetcher"
	_ "github.com/ev1lm0nk3y/gumshoe/http"
	"github.com/ev1lm0nk3y/gumshoe/irc"
	"github.com/ev1lm0nk3y/gumshoe/matcher"
)

func TestLoadUserOrDefaultConfig(t *testing.T) {
	type args struct {
		c string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		if err := LoadUserOrDefaultConfig(tt.args.c); (err != nil) != tt.wantErr {
			t.Errorf("%q. LoadUserOrDefaultConfig() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestGumshoe_setupLogging(t *testing.T) {
	type fields struct {
		tc *config.TrackerConfig
		ic *irc.IrcClient
		ma *matcher.Matcher
		ff *fetcher.FileFetch
		lg *log.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		g := &Gumshoe{
			tc: tt.fields.tc,
			ic: tt.fields.ic,
			ma: tt.fields.ma,
			ff: tt.fields.ff,
			lg: tt.fields.lg,
		}
		if err := g.setupLogging(); (err != nil) != tt.wantErr {
			t.Errorf("%q. Gumshoe.setupLogging() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestGumshoe_Director(t *testing.T) {
	type fields struct {
		tc *config.TrackerConfig
		ic *irc.IrcClient
		ma *matcher.Matcher
		ff *fetcher.FileFetch
		lg *log.Logger
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		g := &Gumshoe{
			tc: tt.fields.tc,
			ic: tt.fields.ic,
			ma: tt.fields.ma,
			ff: tt.fields.ff,
			lg: tt.fields.lg,
		}
		g.Director()
	}
}

func TestGumshoe_startWatchers(t *testing.T) {
	type fields struct {
		tc *config.TrackerConfig
		ic *irc.IrcClient
		ma *matcher.Matcher
		ff *fetcher.FileFetch
		lg *log.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		g := &Gumshoe{
			tc: tt.fields.tc,
			ic: tt.fields.ic,
			ma: tt.fields.ma,
			ff: tt.fields.ff,
			lg: tt.fields.lg,
		}
		if err := g.startWatchers(); (err != nil) != tt.wantErr {
			t.Errorf("%q. Gumshoe.startWatchers() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestGumshoeInit(t *testing.T) {
	type args struct {
		tc *config.TrackerConfig
	}
	tests := []struct {
		name    string
		args    args
		want    *Gumshoe
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		got, err := GumshoeInit(tt.args.tc)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. GumshoeInit() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. GumshoeInit() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_main(t *testing.T) {
	tests := []struct {
		name string
	}{
	// TODO: Add test cases.
	}
	for range tests {
		main()
	}
}
