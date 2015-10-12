package gumshoe

import (
  "testing"

  "github.com/stretchr/testify/assert"
)

func TestAddEpisode(t *testing.T) {
  ne := newEpisode(int64(1), "Episode Test", 1, 2)
  assert.NoError(t, ne.AddEpisode())
  de := newDaily(int64(3), "Daily Test", "2015.01.01")
  assert.NoError(t, de.AddEpisode())
}

func TestGetDailyEpisodesByShowID(t *testing.T) {
  e, err := GetEpisodesByShowID(int64(3))
  assert.NoError(t, err)
  assert.Len(t, *e, 1, "Length of results does not match 1.")
  actual := *e
  assert.Equal(t, "Daily Test", actual[0].Title, "Title does not match.")
  assert.Equal(t, "2015.01.01", actual[0].AirDate, "Airdate doesn't match.")
}

func TestIsNewEpisode(t *testing.T) {
  teste := &Episode{
    ShowID: int64(1),
    Season: 1,
    Episode: 2,
  }
  assert.False(t, teste.IsNewEpisode())

  teste.Episode = 3
  assert.True(t, teste.IsNewEpisode())
}


func TestValidEpisodeQuality(t *testing.T) {
  etest := &Episode{ShowID: int64(3)}
  assert.True(t, etest.ValidEpisodeQuality("show.name.s01e02.1080p.hdtv.mp4.torrent"))
  assert.False(t, etest.ValidEpisodeQuality("show.name.s01e02.HDTV.mp4.torrent"))

  etest.ShowID = int64(1)
  assert.True(t, etest.ValidEpisodeQuality("show.name.s01e02.720p.hdtv.mp4.torrent"))
  assert.False(t, etest.ValidEpisodeQuality("show.name.s01e02.HDTV.mp4.torrent"))

  etest.ShowID = int64(2)
  assert.True(t, etest.ValidEpisodeQuality("show.name.s01e02.HDTV.mp4.torrent"))
  assert.False(t, etest.ValidEpisodeQuality("show.name.s01e02.720p.hdtv.mp4.torrent"))
}


func TestParseTorrentString(t *testing.T) {
  eTest, err := ParseTorrentString("daily.shown.2015.03.26.will.ferrell.1080p.hdtv.x264-daview.mp4.torrent")
  assert.NoError(t, err)
  expected := newDaily(int64(3), "Will Ferrell", "2015.03.26")
  assert.Equal(t, expected, eTest, "Objects don't match")

  eTest, err = ParseTorrentString("Walking.bread.S01E06.In.Enemy.Hands.1080p.WEB-DL.DD5.1.H.264-NTb.torrent")
  assert.NoError(t, err)
  expected = newEpisode(int64(1), "In Enemy Hands", 1, 6)
  assert.Equal(t, expected, eTest, "Episode objects don't match")

  _, err = ParseTorrentString("blah.blahblah.not.a.real.episode.torrent")
  assert.Error(t, err)

  _, err = ParseTorrentString("the.thundermans.s02e22.one.hit.thunder.hdtv.x264-w4f.mp4.torrent")
  PrintDebugln(err)
  assert.Error(t, err)
}

func TestEpisodeRewriter(t *testing.T) {
  assert.Equal(t, "This Is A Title", episodeRewriter("this.is.a.title"))
}
