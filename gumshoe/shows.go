/* TV Show and Episode Tracker
 *
 * Add, delete and list shows and episodes that you keep track of with this
 * component. The section labeled "User Functions" lists the actions that you
 * can perform with the data.
 *
 * The remainder of the code takes input from your sources, parses them and
 * compares the title, episode number and quality against the watchlist.
 * Also keeps track of seen episodes to prevent duplicates.
 */
package gumshoe

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/coopernurse/gorp"
)

var showDB *gorp.DbMap // Database

type Show struct {
	ID         int64     `json:"ID,omitempty"`
	Title      string    `json:"title" binding:"required"`
	Quality    string    `json:"quality"`
	Episodal   bool      `json:"episodal"`
	LastUpdate time.Time `json:"last_update"`
}

type Shows struct {
	Shows []Show
}

type Episode struct {
	ID      int64     `json:"id"`
  ShowID  int64     `json:"show_id" binding:"required"`
	Title   string    `json:"title" binding:"required"`
	Season  int       `json:"season"`
	Episode int       `json:"episode"`
	AirDate string    `json:"airdate"`
	Added   time.Time `json:"added"`
}

type Episodes struct {
  Episodes []Episode
}

func newShow(t string, q string, e bool) *Show {
	return &Show{
		Title:      t,
		Quality:    q,
		Episodal:   e,
		LastUpdate: time.Now(),
	}
}

func newEpisode(sid int64, t string, s, e int) *Episode {
	return &Episode{
    ShowID:  sid,
		Title:   t,
		Season:  s,
		Episode: e,
		Added:   time.Now(),
	}
}

func newDaily(sid int64, t, d string) *Episode {
	return &Episode{
    ShowID:  sid,
		Title:   t,
		AirDate: d,
		Added:   time.Now(),
	}
}

func InitShowDb(baseDir string) {
	showDB = initDb(baseDir, "shows")
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

/*
 * Begin User Functions
 */
func AddEpisode(episode *Episode) error {
	err := showDB.Insert(episode)
	return err
}

func GetEpisodesByShowID(id int64) (Episodes, error) {
  var allE Episodes
  e := Episode{ShowID: id}
  _, err := showDB.Select(&allE.Episodes, "select * from episode where ShowID=?", e.ShowID)
  return allE, err
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
/*
 * End User Functions
 */


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
	if err != nil {
		return err
	}

	tvShow.LastUpdate = time.Now()
	if !verifyQuality(&tvShow, e[3]) {
		return errors.New(fmt.Sprintf("No quality match for %s\n", e[0]))
	}

	switch len(e) {
	case 6:
		episode, err := unseenEpisode(&tvShow, showTitle, e[3], e[4])
		if err != nil {
			return err
		}
		AddEpisode(episode)
	case 4:
		episode, err := unseenDaily(&tvShow, showTitle, e[2])
		if err != nil {
			return err
		}
		AddEpisode(episode)
	default:
		return errors.New(fmt.Sprintf("Episode string invalidly parsed: %s", e))
	}
	return nil
}

func unseenEpisode(show *Show, t, s, e string) (*Episode, error) {
	if show.Episodal {
		eCheck := newEpisode(show.ID, t, GetInt(s), GetInt(e))
		err := showDB.SelectOne(&eCheck,
			"select * from episodes where Title=? and Season=? and Episode=?",
			t, GetInt(s), GetInt(e))
		return eCheck, err
	}
	return nil, errors.New("Something went wrong with the regex match.")
}

func unseenDaily(show *Show, t, eDetails string) (*Episode, error) {
	if !show.Episodal {
		dRegexp, _ := regexp.Compile("^.*(\\d{4}\\.\\d{2}\\.\\d{2}).+$")
		date := dRegexp.FindString(eDetails)
		if date != "" {
			eCheck := newDaily(show.ID, t, date)
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
  // TODO: TV show quality will change over the years, make this more maintainable.
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
