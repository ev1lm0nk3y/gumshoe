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

import "time"

type Show struct {
	ID         int64     `json:"ID,omitempty"`
	Title      string    `json:"title" binding:"required"`
	Quality    string    `json:"quality"`
	Episodal   bool      `json:"episodal"`
	LastUpdate int64     `json:"last_update"`
}

func newShow(t, q string, e bool) *Show {
	return &Show{
		Title:      episodeRewriter(t),
		Quality:    q,
		Episodal:   e,
		LastUpdate: time.Now().UnixNano(),
	}
}

func (s *Show) AddShow() error {
	err := gDb.Insert(s)
	return err
}

func (s *Show) DeleteShow() error {
	_, err := gDb.Exec("delete from show where ID=?", s.ID)
	return err
}

func (s *Show) UpdateShow() error {
	_, err := gDb.Update(s)
	return err
}

func ListShows() (*[]Show, error) {
	shows := &[]Show{}
	_, err := gDb.Select(shows, "select * from show order by Title")
	return shows, err
}

func GetShow(id int64) (Show, error) {
	show := Show{}
	err := gDb.SelectOne(&show, "select * from show where ID=?", id)
	return show, err
}

func GetShowByTitle(title string) (Show, error) {
	show := Show{}
	err := gDb.SelectOne(&show, "select * from show where Title=?", episodeRewriter(title))
	return show, err
}
