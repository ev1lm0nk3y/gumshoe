/* TV Show Tracker
 *
 * Add, delete and list shows that you keep track of with this component.
 * The section labeled "User Functions" lists the actions that you can perform with the data.
 */
package db

import (
  "time"

  //"github.com/garfunkel/go-tvdb"
)

type Show struct {
	ID         int64  `json:"ID,omitempty"`
  TvDbId     uint64 `json:"tvdbid"`
	Title      string `json:"title" binding:"required"`
	Quality    string `json:"quality"`
	Episodal   bool   `json:"episodal"`
	LastUpdate int64  `json:"last_update"`
}

func NewShow(t, q string, e bool) *Show {
	return &Show{
		Title:      episodeRewriter(t),
		Quality:    q,
		Episodal:   e,
		LastUpdate: time.Now().UnixNano(),
	}
}

func (s *Show) AddShow() error {
	err := gDb.Insert(s)
  go AddDBOp("show")
	return err
}

func (s *Show) DeleteShow() error {
	_, err := gDb.Exec("delete from show where ID=?", s.ID)
  go AddDBOp("show")
	return err
}

func (s *Show) UpdateShow() error {
  checkDBLock<- 1
	_, err := gDb.Update(s)
  <-checkDBLock
  go AddDBOp("show")
	return err
}

func ListShows() ([]Show, error) {
	shows := []Show{}
	_, err := gDb.Select(&shows, "select * from show order by Title")
  go AddDBOp("show")
	return shows, err
}

func GetShow(id int64) (Show, error) {
	show := Show{}
	err := gDb.SelectOne(&show, "select * from show where ID=?", id)
  go AddDBOp("show")
	return show, err
}

func GetShowByTitle(title string) (Show, error) {
	show := Show{}
	err := gDb.SelectOne(&show, "select * from show where Title like '%%?%%'", episodeRewriter(title))
  go AddDBOp("show")
	return show, err
}
