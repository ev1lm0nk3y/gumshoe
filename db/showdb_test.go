package db

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testDataDir string

func TestMain(m *testing.M) {
	flag.Parse()
	tc = NewTrackerConfig()
	tc.LoadGumshoeConfig(configFile)
	tc.Operations.Debug = true

	err := InitDb()
	if err != nil {
		log.Fatalln("DB Init Failed.")
	}
	err = gDb.TruncateTables()
	if err != nil {
		log.Println("Clearing DB failed.", err)
	}
	s := newShow("walking bread", "720p", true)
	err = s.AddShow()
	if err != nil {
		log.Println("InitDb:AddShow:err", err)
	}
	s = newShow("Game of Chowns", "420", true)
	err = s.AddShow()
	if err != nil {
		log.Println("InitDb:AddShow:err", err)
	}
	s = newShow("Daily Shown", "1080p", false)
	err = s.AddShow()
	if err != nil {
		log.Println("InitDb:AddShow:err", err)
	}
	m.Run()

	gDb.DropTablesIfExists()

	if testDataDir == "" {
		wd, _ := os.Getwd()
		testDataDir = filepath.Join(wd, "test_data")
	}
	os.Remove(filepath.Join(testDataDir, "gumshoe.db"))
	os.Remove(filepath.Join(testDataDir, "conan.2015.03.26.will.ferrell.hdtv.x264-daview.mp4.torrent"))
	os.Remove(filepath.Join(testDataDir, "the.thundermans.s02e22.one.hit.thunder.hdtv.x264-w4f.mp4.torrent"))
	os.Remove(filepath.Join(testDataDir, "the.thundermans.s02e23.the.girl.with.the.dragon.snafu.hdtv.x264-w4f.mp4.torrent"))
	os.Remove(filepath.Join(testDataDir, "X.Company.S01E06.In.Enemy.Hands.1080p.WEB-DL.DD5.1.H.264-NTb.torrent"))
	os.Exit(0)
}

func TestListShows(t *testing.T) {
	allShows, err := ListShows()
	assert.NoError(t, err)
	for _, i := range allShows {
		assert.IsType(t, "test", i.Title)
		assert.NotEmpty(t, i.ID, "IDs were not automatically added to %s.", i.Title)
		assert.NotEqual(t, i.ID, 0, "ID numbers should never be 0.")
	}
}

func TestGetShow(t *testing.T) {
	sid := int64(1)
	expected := &Show{
		ID:       sid,
		Title:    "Walking Bread",
		Quality:  "720p",
		Episodal: true,
	}
	s, err := GetShow(sid)
	// Ensure we don't fail because the time is off
	s.LastUpdate = int64(0)
	assert.NoError(t, err)
	if !assert.ObjectsAreEqual(expected, &s) {
		t.Errorf("show objects do not match.\nExpected: %s\nActual: %s\n", expected, s)
	}
}

func TestGetShowByTitle(t *testing.T) {
	expected := &Show{
		ID:       int64(3),
		Title:    "Daily Shown",
		Quality:  "1080p",
		Episodal: false,
	}
	s, err := GetShowByTitle("Daily Shown")
	// Ensure we don't fail because the time is off
	s.LastUpdate = int64(0)
	assert.NoError(t, err)
	if !assert.ObjectsAreEqual(expected, &s) {
		t.Errorf("show objects do not match.\nExpected: %s\nActual: %s\n", expected, s)
	}
}
