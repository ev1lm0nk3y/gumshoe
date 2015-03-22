/* Package watcher
 * Program that takes episode listings in, parses them and compares the title, episode number and
 * quality against the watchlist.
 */
package gumshoe

import (
	"errors"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coopernurse/gorp"
)

var showDB *gorp.DbMap

type Show struct {
	ID       int64  `json:"ID,omitempty"`
	Title    string `json:"title" binding:"required"`
	Quality  string `json:"quality"`
	Episodal bool   `json:"episodal"`
}

type Shows struct {
	Shows []Show
}

type Episode struct {
	ID      int64     `json:"ID"`
	Title   string    `json:"title" binding:"required"`
	Season  int       `json:"season"`
	Episode int       `json:"episode"`
	AirDate string    `json:"airdate"`
	Added   time.Time `json:"added"`
}

func newShow(t string, q string, e bool) *Show {
	return &Show{
		Title:    t,
		Quality:  q,
		Episodal: e,
	}
}

func newEpisode(t string, s, e int) *Episode {
	return &Episode{
		Title:   t,
		Season:  s,
		Episode: e,
		Added:   time.Now(),
	}
}

func newDaily(t, d string) *Episode {
	return &Episode{
		Title:   t,
		AirDate: d,
		Added:   time.Now(),
	}
}

func InitShowDb(baseDir string) {
	showDB = initDb(baseDir, "shows")
	//TODO(deekue) is this needed?
	// defer showDB.Db.Close()
	initTable(showDB, Show{}, "shows")
	initTable(showDB, Episode{}, "episodes")
}

func LoadTestData() {
	var quality string
	// delete any existing rows
	err := showDB.TruncateTables()
	checkErr(err, "TruncateTables failed")

	err = AddShow("walking bread", quality, true)
	log.Println("InitShowDb:AddShow:err", err)

	err = AddShow("game of chowns", quality, true)
	log.Println("InitShowDb:AddShow:err", err)

	err = AddShow("daily shown", quality, false)
	log.Println("InitShowDb:AddShow:err", err)
}

func AddEpisode(episode *Episode) error {
	err := showDB.Insert(episode)
	return err
}

func ListShows() (Shows, error) {
	var shows Shows
	_, err := showDB.Select(&shows.Shows, "select * from shows order by Title")
	return shows, err
}

func GetShow(id int64) (Show, error) {
	show := Show{ID: id}
	err := showDB.SelectOne(&show, "select * from shows where ID=?", show.ID)
	return show, err
}

func GetShowByTitle(title string) (Show, error) {
	show := Show{Title: title}
	err := showDB.SelectOne(&show, "select * from shows where Title=?", show.Title)
	return show, err
}

func AddShow(t string, q string, e bool) error {
	show := newShow(t, q, e)
	err := showDB.Insert(show)
	return err
}

func DeleteShow(show Show) error {
	_, err := showDB.Exec("delete from shows where ID=?", show.ID)
	return err
}

func UpdateShow(show Show) error {
	_, err := showDB.Update(&show)
	return err
}

func episodeRewriter(ep string) string {
	// strip show title of "." and make it Title cased, easier to do string matching
	e := strings.Replace(ep, ".", " ", -1)
	return strings.Title(e)
}

func IsNewEpisode(e []string) error {
  // the string slice should be as follows:
	// [ "whole string match", "show title", "full episode desc", "season #", "episode #",
	//   "remainder" ]
	// Though if there are no season or episode numbers, this could mean that it is a daily show.
	showTitle := episodeRewriter(e[1])
	tvShow, err := GetShowByTitle(showTitle)
	if err == nil {
		if len(e) == 6 {
			episode, err := unseenEpisode(&tvShow, showTitle, e[3], e[4])
			if err == nil && verifyQuality(&tvShow, e[3]) {
				AddEpisode(episode)
			}
			return err
		}
		if len(e) == 4 {
			episode, err := unseenDaily(&tvShow, showTitle, e[2])
			if err == nil && verifyQuality(&tvShow, e[3]) {
				AddEpisode(episode)
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

func unseenEpisode(show *Show, t, s, e string) (*Episode, error) {
	if show.Episodal {
		eCheck := newEpisode(t, getInt(s), getInt(e))
		err := showDB.SelectOne(&eCheck,
			"select * from episodes where Title=? and Season=? and Episode=?",
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
			err := showDB.SelectOne(&eCheck,
				"select * from episodes where Title=? and AirDate=?",
				t, date)
			return eCheck, err
		}
		return nil, errors.New("No date match found in episode details")
	}
	return nil, errors.New("This episode was matched wrong." + t)
}

func verifyQuality(show *Show, s string) bool {
  hdtvRegexp, _ := regexp.Compile("(720|1080)[ip]")
  isHDTV := hdtvRegexp.MatchString(s)
  if show.Quality == "420" && isHDTV {
    return false
  } else {
    hdtvQuality := hdtvRegexp.FindStringSubmatch(s)[1]
    if show.Quality != hdtvQuality {
      return false
    }
  }
	return true
}
