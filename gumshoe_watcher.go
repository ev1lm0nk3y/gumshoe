/* Gumshoe Watcher
 * Program that takes episode listings in, parses them and compares the title, episode number and
 * quality against the watchlist.
 */
package gumshoe

import (
  "database/sql"
  "fmt"
  "github.com/coppernurse/gorp"
  _ "github.com/mattn/go-sqlite3"
  "log"
  "time"
)

type Episode struct {
  title     string
  season    int
  episode   int
  added     time.Time
}

func init() {
  // Metrics

  // Database
  db := initDb(tc.Files['episode_state'])
  defer db.Db.Close()
}

func initDb(dbLocation string) *gorp.DbMap {
  dbConn, err := sql.Open("sqlite3", dbLocation)
  checkErr(err, "Database unable to be opened.")

  dbMap := &gorp.DbMap{Db: dbConn, Dialect: gorp.SqliteDialect{}}
  dbMap.AddTableWithName(Episode{}, "episodes")
  err = dbMap.CreateTablesIfNotExists()
  checkErr(err, "Unable to create episode tables in DB.")

  return dbMap
}

func newEpisode(t string, s, e int) Episode {
  return Episode{
    title: t,
    season: s,
    episode: e,
    added: time.Now().UnixNano()
  }
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

func IsNewEpisode(e []string) bool {
  // the string slice should be as follows
  // [ "whole string match", "show title", "full episode desc", "season #", "episode #",
  //   "remainder" ]
  if len(e) != 6 {
    return false
  }
  showTitle = episodeRewriter(e[1])
  _, tvShow, err := GetShow(showTitle)
  if err != nil {
    // log line?
    return false
  }
  // log that we found a show
  if unseenEpisode(tvShow, showTitle, e[3], e[4]) {
    if verifyQuality(tvShow, e[5]) {
      addEpisodeToDb(showTitle, e[3], e[4])
      return true
    }
  }
  return false
}

func unseenEpisode(show *Show, t,s,e string) bool {
  if show.Episodal {
    eCheck := newEpisode(t, int(s), int(e))
    err := dbMap.SelectOne(&eCheck,
                           "select * from episodes where title=? and season=? and episode=?",
                           t, int(s), int(e))
    if err != nil {
      // log that this is an unseen episode
      return true
    }
  } else {
    // how do i handle this situation?




	if ap := shows.AnnounceLine.FindStringSubmatch(string.toLower(e.Raw)); ap != nil {
		for p := range shows.Shows {
			if match, _ := regexp.MatchString(p.Title, ap[1]); match {
				log.Println("Episode match found. %s", e.Raw)
				if p.EpisodeOnly {
					if shows.EpisodeShow.MatchString(ap[1]) {
						log.Println("This is an episode.")
					} else {
						log.Println("Not interested.")
						return
					}
				}
				if !shows.ExcludeShows.MatchString(ap[1]) {
					if !downloader.CheckPreviousEpisodes(ap[1]) {
						log.Println("Full match, grabbing.")
						downloader.GetEpisode(ap[2])
					}
				}
			}
		}
	}
