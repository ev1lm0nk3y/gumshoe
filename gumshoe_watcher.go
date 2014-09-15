/* Gumshoe Watcher
 * Program that takes episode listings in, parses them and compares the title, episode number and
 * quality against the watchlist.
 */
package gumshoe

import (
  "database/sql"
  "errors"
  "github.com/coopernurse/gorp"
  _ "github.com/mattn/go-sqlite3"
  "log"
  "regexp"
  "strconv"
  "strings"
  "time"
)

var db *EpisodeDB

type Episode struct {
  title     string
  season    int
  episode   int
  airDate   string
  added     time.Time
}

func newEpisode(t string, s, e int) *Episode {
  return &Episode{
    title: t,
    season: s,
    episode: e,
    added: time.Now(),
  }
}

func newDaily(t, d string) *Episode {
  return &Episode{
    title: t,
    airDate: d,
    added: time.Now(),
  }
}

type EpisodeDB struct {
  location   string
  conn       *gorp.DbMap
}

func initWatcher(tc *TrackerConfig) {
  // Metrics

  // Database
  db := new(EpisodeDB)
  db.location = tc.Files["episode_state"]
  db.Init()
  defer db.conn.Db.Close()
}

func (db *EpisodeDB) Init() {
  dbConn, err := sql.Open("sqlite3", db.location)
  checkErr(err, "Database unable to be opened.")

  dbMap := &gorp.DbMap{Db: dbConn, Dialect: gorp.SqliteDialect{}}
  dbMap.AddTableWithName(Episode{}, "episodes")
  err = dbMap.CreateTablesIfNotExists()
  checkErr(err, "Unable to create episode tables in DB.")

  db.conn = dbMap
}

func (db *EpisodeDB) addEpisodeToDb(episode *Episode) error {
  err := db.conn.Insert(episode)
  return err
}

// this func should be moved to another file
func checkErr(err error, msg string) {
  if err != nil {
    log.Fatalln(msg, err)
  }
}

func episodeRewriter(ep string) string {
  // strip show title of "." and make it lowercase, easier to do string matching
  e := strings.Replace(ep, ".", " ", -1)
  return strings.Title(e)
}

func IsNewEpisode(e []string) error {
  // the string slice should be as follows
  // [ "whole string match", "show title", "full episode desc", "season #", "episode #",
  //   "remainder" ]
  // Though if there are no season or episode numbers, this could mean that it is a daily show.
  showTitle := episodeRewriter(e[1])
  _, tvShow, err := allShows.GetShow(showTitle)
  if err == nil {
    if len(e) == 6 {
      episode, err := unseenEpisode(tvShow, showTitle, e[3], e[4])
      if err == nil && verifyQuality(tvShow, e[3]) {
        db.addEpisodeToDb(episode)
      }
      return err
    }
    if len(e) == 4 {
      episode, err := unseenDaily(tvShow, showTitle, e[2])
      if err == nil && verifyQuality(tvShow, e[3]) {
        db.addEpisodeToDb(episode)
      }
      return err
    }
  }
  return err
}

func getInt(s string) int {
  r, _ := strconv.Atoi(s)
  return r
}

func unseenEpisode(show *Show, t,s,e string) (*Episode, error) {
  if show.Episodal {
    eCheck := newEpisode(t, getInt(s), getInt(e))
    err := db.conn.SelectOne(&eCheck,
                             "select * from episodes where title=? and season=? and episode=?",
                             t, getInt(s), getInt(e))
    return eCheck, err
  }
  return nil, errors.New("Something went wrong with the regex match.")
}

func unseenDaily(show *Show, t, eDetails string) (*Episode, error) {
  if !show.Episodal {
    dRegexp, _ := regexp.Compile("^.*(\\d{4}\\.\\d{2}\\.\\d{2}).+$")
    date := dRegexp.FindString(eDetails)
    if date != "" {
      eCheck := newDaily(t, date)
      err := db.conn.SelectOne(&eCheck,
                               "select * from episodes where title=? and airDate=?",
                               t, date)
      return eCheck, err
    }
    return nil, errors.New("No date match found in episode details")
  }
  return nil, errors.New("This episode was matched wrong." + t)
}

func verifyQuality(show *Show, s string) bool {
  return true
}
