/* Episode Database
 *
 * Gumshoe's episode database structs and the functions that modify it.
 * Add, delete and list your episodes that you keep track of with this
 * component.
 */
package db

import (
	"errors"
	"fmt"
  "net/url"
  "regexp"
	"strings"
	"time"

  "github.com/ev1lm0nk3y/gumshoe/misc"
)

var (
  episodeQualityRegexp *regexp.Regexp
  episodePattern *regexp.Regexp
)

type Episode struct {
	ID      int64  `json:"id"`
	ShowID  int64  `json:"show_id" binding:"required"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	AirDate string `json:"airdate"`
	Added   int64  `json:"added"`
}

func newEpisode(sid int64, s, e int) *Episode {
	return &Episode{
		ShowID:  sid,
		Season:  s,
		Episode: e,
		Added:   time.Now().UnixNano(),
	}
}

func newDaily(sid int64, t, d string) *Episode {
	return &Episode{
		ShowID:  sid,
		AirDate: d,
		Added:   time.Now().UnixNano(),
	}
}

// Start User Functions
func (e *Episode) AddEpisode() (err error) {
	e.Added = time.Now().UnixNano()
  checkDBLock<- 1
  err = gDb.Insert(e)
  <-checkDBLock
  show, err := GetShow(e.ShowID)
  if err == nil {
    show.LastUpdate = e.Added
    err = show.UpdateShow()
  }
  go AddDBOp("episode")
	return err
}

func (e *Episode) IsNewEpisode() bool {
  checkDBLock<- 1
	err := gDb.SelectOne(&Episode{}, "select ID from episode where ShowID=? and Season=? and Episode=? and AirDate=?",
		e.ShowID, e.Season, e.Episode, e.AirDate)
  <-checkDBLock
  go AddDBOp("episode")
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

func GetEpisodesByShowID(id int64) (allE *[]Episode, err error) {
  checkDBLock<- 1
	_, err = gDb.Select(allE, "select * from episode where ShowID=?", id)
  <-checkDBLock
  go AddDBOp("episode")
	return
}

func GetLastEpisode(sid int64) (le *Episode, err error) {
  checkDBLock<- 1
  err = gDb.SelectOne(le, "select * from episode where ShowID=? sort by AirDate, Season, Episode desc limit 1", sid)
  <-checkDBLock
  go AddDBOp("episode")
  return
}

func ParseTorrentString(e string) (episode *Episode, err error) {
	eMatch, err := matchEpisodeToPattern(e)
	if err != nil {
		return
	}
	sid, err := GetShowByTitle(episodeRewriter(eMatch["show"]))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Show %s is not being tracked.", episodeRewriter(eMatch["show"])))
	}
	episode.ShowID = sid.ID

	episode.Season = misc.GetInt(eMatch["season"])
	episode.Episode = misc.GetInt(eMatch["episode"])
  episode.AirDate = eMatch["airdate"]
  if eMatch["enum"] != "" {
    episode.Season = misc.GetInt(string(eMatch["enum"][0]))
    episode.Episode = misc.GetInt(string(eMatch["enum"][1:]))
  }
	return
}

// End User Functions

func episodeRewriter(ep string) string {
	e := strings.Replace(ep, ".", " ", -1)
	return strings.Title(e)
}

func matchEpisodeToPattern(e string) (named map[string]string, err error) {
  match := episodePattern.FindAllStringSubmatch(e, -1)
  if match == nil {
    return nil, errors.New("string not matched regexp")
  }

  for i, n := range match[0] {
    named[episodePattern.SubexpNames()[i]] = n
  }
  return
}

func CheckMatch(m string) (*Episode, error) {
  ep, err := ParseTorrentString(m)
  if err != nil {
    return nil, errors.New("No episode regexp match.")
  }
  isNew := ep.IsNewEpisode()
  if !isNew {
    return nil, errors.New("Episode exists.")
  }
  if !ep.ValidEpisodeQuality(m) {
    return nil, errors.New("Wrong Episode Quality.")
  }
  return ep, nil
}

func SetEpisodeQualityRegexp(r string) (err error) {
  episodeQualityRegexp, err = regexp.Compile(r)
  return err
}

func SetEpisodePatternRegexp(r string) (err error) {
  ue, err := url.QueryUnescape(r)
  if err != nil {
    return err
  }
  episodePattern, err = regexp.Compile(ue)
  return err
}
