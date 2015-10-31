/* Episode Database
 *
 * Gumshoe's episode database structs and the functions that modify it.
 * Add, delete and list your episodes that you keep track of with this
 * component.
 */
package gumshoe

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Episode struct {
	ID      int64  `json:"id"`
	ShowID  int64  `json:"show_id" binding:"required"`
	Title   string `json:"title" binding:"required"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	AirDate string `json:"airdate"`
	Added   int64  `json:"added"`
}

func newEpisode(sid int64, t string, s, e int) *Episode {
	return &Episode{
		ShowID:  sid,
		Title:   episodeRewriter(t),
		Season:  s,
		Episode: e,
		Added:   time.Now().UnixNano(),
	}
}

func newDaily(sid int64, t, d string) *Episode {
	return &Episode{
		ShowID:  sid,
		Title:   episodeRewriter(t),
		AirDate: d,
		Added:   time.Now().UnixNano(),
	}
}

// Start User Functions
func (e *Episode) AddEpisode() error {
	e.Added = time.Now().UnixNano()
  checkDBLock<- 1
  err := gDb.Insert(e)
  <-checkDBLock
	return err
}

func (e *Episode) IsNewEpisode() bool {
  checkDBLock<- 1
	err := gDb.SelectOne(&Episode{}, "select ID from episode where ShowID=? and Season=? and Episode=? and AirDate=?",
		e.ShowID, e.Season, e.Episode, e.AirDate)
  <-checkDBLock
	if err == nil {
		return false
	}
	return true
}

func (e *Episode) ValidEpisodeQuality(s string) bool {
	isHDTV := episodeQualityRegexp.MatchString(s)
	show, _ := GetShow(e.ShowID)
	if show.Quality == "420" || show.Quality == "" {
		return !isHDTV
	} else {
		return show.Quality == episodeQualityRegexp.FindString(s)
	}
}

func GetEpisodesByShowID(id int64) (*[]Episode, error) {
	allE := &[]Episode{}
  checkDBLock<- 1
	_, err := gDb.Select(allE, "select * from episode where ShowID=?", id)
  <-checkDBLock
	return allE, err
}

func ParseTorrentString(e string) (*Episode, error) {
	eMatch := episodePattern.FindStringSubmatch(e)
	if eMatch == nil {
		return nil, errors.New(fmt.Sprintf("This isn't an episode: %s", e))
	}
	st := episodeRewriter(eMatch[1])
	sid, err := GetShowByTitle(st)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Show %s is not being tracked.", st))
	}
	episode := &Episode{
		ShowID: sid.ID,
		Title:  eMatch[7],
	}
	if eMatch[4] != "" {
		episode.AirDate = eMatch[4]
	} else {
		if eMatch[2] != "" {
			episode.Season = GetInt(eMatch[2])
			episode.Episode = GetInt(eMatch[3])
		} else {
			episode.Season = GetInt(eMatch[5])
			episode.Episode = GetInt(eMatch[6])
		}
	}
	return episode, nil
}

// End User Functions

func episodeRewriter(ep string) string {
	// strip show title of "." and make it Title cased, easier to do string matching
	e := strings.Replace(ep, ".", " ", -1)
	return strings.Title(e)
}
