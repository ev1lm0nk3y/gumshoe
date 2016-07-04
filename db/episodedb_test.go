package db

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func EpisodeTestMain(m *testing.M) {
	if err := InitDb("test_edb"); err != nil {
		log.Fatalf("DB Init Failed. %s", err)
	}

	if err := gDb.TruncateTables(); err != nil {
		log.Fatalf("Clearing DB failed. %s", err)
	}
	if err := NewShow("test title", "", true).AddShow(); err != nil {
		log.Fatalf("1st show addition failed. %s", err)
	}
	if err := NewShow("test title2", "720p", true).AddShow(); err != nil {
		log.Fatalf("2nd show addition failed. %s", err)
	}
	if err := NewShow("daily show", "1080p", false).AddShow(); err != nil {
		log.Fatalf("3rd show addition failed. %s", err)
	}
	m.Run()
}

func TestAddEpisode(t *testing.T) {
	ne := NewEpisode(int64(1), 1, 2)
	assert.NoError(t, ne.AddEpisode())
	de := NewDaily(int64(3), "2015.01.01")
	assert.NoError(t, de.AddEpisode())
}

func TestGetDailyEpisodesByShowID(t *testing.T) {
	e, err := GetEpisodesByShowID(int64(3))
	assert.NoError(t, err)
	assert.Len(t, *e, 1, "Length of results does not match 1.")
	a := *e
	assert.Equal(t, "2015.01.01", a[0].AirDate, "Airdate doesn't match.")
}

func TestIsNewEpisode(t *testing.T) {
	teste := &Episode{
		ShowID:  int64(1),
		Season:  1,
		Episode: 2,
	}
	assert.False(t, teste.IsNewEpisode())

	teste.Episode = 3
	assert.True(t, teste.IsNewEpisode())
}

func TestValidEpisodeQuality(t *testing.T) {
	err := SetEpisodeQualityRegexp("1080p|720p")
	assert.NoError(t, err, "quality regexp didn't parse corretly.")
	etest := &Episode{ShowID: int64(3)}
	assert.True(t, etest.ValidEpisodeQuality("show.name.s01e02.1080p.hdtv.mp4.torrent"))
	assert.False(t, etest.ValidEpisodeQuality("show.name.s01e02.HDTV.mp4.torrent"))

	etest.ShowID = int64(1)
	assert.True(t, etest.ValidEpisodeQuality("show.name.s01e02.720p.hdtv.mp4.torrent"))
	assert.False(t, etest.ValidEpisodeQuality("show.name.s01e02.HDTV.mp4.torrent"))

	etest.ShowID = int64(2)
	err = SetEpisodeQualityRegexp("420|HDTV")
	assert.True(t, etest.ValidEpisodeQuality("show.name.s01e02.HDTV.mp4.torrent"))
	assert.False(t, etest.ValidEpisodeQuality("show.name.s01e02.720p.hdtv.mp4.torrent"))
}

func TestParseTorrentString(t *testing.T) {
	err := SetEpisodePatternRegexp("%5E(%5B%5Cw%5Cd%5Cs.%5D%2B)%5B.%20%5D(%3F%3As(%5Cd%7B1%2C2%7D)e(%5Cd%7B1%2C2%7D)%7C(%5Cd)x%3F(%5Cd%7B2%7D))%5B.%20%5D")
	eTest, err := ParseTorrentString("daily.shown.2015.03.26.will.ferrell.1080p.hdtv.x264-daview.mp4.torrent")
	assert.NoError(t, err)
	expected := NewDaily(int64(3), "2015.03.26")
	assert.Equal(t, expected, eTest, "Objects don't match")

	eTest, err = ParseTorrentString("Walking.bread.S01E06.In.Enemy.Hands.1080p.WEB-DL.DD5.1.H.264-NTb.torrent")
	assert.NoError(t, err)
	expected = NewEpisode(int64(1), 1, 6)
	assert.Equal(t, expected, eTest, "Episode objects don't match")

	_, err = ParseTorrentString("blah.blahblah.not.a.real.episode.torrent")
	assert.Error(t, err)

	_, err = ParseTorrentString("the.thundermans.s02e22.one.hit.thunder.hdtv.x264-w4f.mp4.torrent")
	assert.Error(t, err)
}

func TestEpisodeRewriter(t *testing.T) {
	assert.Equal(t, "This Is A Title", episodeRewriter("this.is.a.title"))
}
