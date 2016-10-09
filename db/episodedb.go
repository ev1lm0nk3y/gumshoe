/* Episode Database
 *
 * Gumshoe's episode database structs and the functions that modify it.
 * Add, delete and list your episodes that you keep track of with this
 * component.
 */
package db

import (
	"regexp"
	"sync"
	"time"
)

var (
	episodeQualityRegexp *regexp.Regexp
	episodePattern       *regexp.Regexp
	edbl                 sync.RWMutex
)

// Episode describes the database fields and how they can map to json output.
// Episodes reference the Show database for the ShowID.
type Episode struct {
	ID      int64  `json:"id"`
	ShowID  int64  `json:"show_id" binding:"required"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	AirDate string `json:"airdate"`
	Added   int64  `json:"added"`
}

// NewEpisode takes a Show ID, season and episode numbers and creates an
// Episode object.
func NewEpisode(sid int64, s, e int) *Episode {
	return &Episode{
		ShowID:  sid,
		Season:  s,
		Episode: e,
		Added:   time.Now().UnixNano(),
	}
}

// NewDaily takes a Show ID and a date string and creates an Episode object for
// daily occuring shows.
func NewDaily(sid int64, d string) *Episode {
	return &Episode{
		ShowID:  sid,
		AirDate: d,
		Added:   time.Now().UnixNano(),
	}
}

// AddEpisode will insert the Episode into the DB and update the Show DB with
// the time it happened.
func (e *Episode) AddEpisode() error {
	e.Added = time.Now().UnixNano()
	edbl.Lock()
	err := gDb.Insert(e)
	edbl.Unlock()
	go AddDBOp("episode")

	show, err := GetShow(e.ShowID)
	if err != nil {
		return err
	}
	show.LastUpdate = e.Added
	return show.UpdateShow()
}

// IsNewEpisode checks the Episode DB against the current object and returns
// true if is an unseen episode.
func (e *Episode) IsNewEpisode() bool {
	edbl.RLock()
	defer edbl.RUnlock()
	err := gDb.SelectOne(&Episode{}, "select ID from episode where ShowID=? and Season=? and Episode=? and AirDate=?",
		e.ShowID, e.Season, e.Episode, e.AirDate)
	go AddDBOp("episode")
	if err == nil {
		return false
	}
	return true
}

// GetEpisodesByShowID will return all the episodes within the DB that match the Show ID.
func GetEpisodesByShowID(id int64) (*[]Episode, error) {
	allE := &[]Episode{}
	edbl.RLock()
	defer edbl.RUnlock()
	_, err := gDb.Select(allE, "select * from episode where ShowID=?", id)
	go AddDBOp("episode")
	return allE, err
}

// GetLastEpisode will return the last Episode entry for the given Show ID.
func GetLastEpisode(sid int64) (*Episode, error) {
	var le *Episode
	edbl.RLock()
	defer edbl.RUnlock()
	err := gDb.SelectOne(le, "select * from episode where ShowID=? sort by AirDate, Season, Episode desc limit 1", sid)
	go AddDBOp("episode")
	return le, err
}
