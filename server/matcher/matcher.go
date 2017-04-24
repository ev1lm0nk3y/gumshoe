//Package matcher is your onestop shop for all log message processing and validation.
package matcher

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"sync"

	"github.com/ev1lm0nk3y/gumshoe/config"
	"github.com/ev1lm0nk3y/gumshoe/misc"
	"github.com/ev1lm0nk3y/gumshoe/server/db"
	"github.com/ev1lm0nk3y/gumshoe/server/fetcher"
)

// MatchState controls the channel on sent messages
type MatchState int

const (
	matchStateList MatchState = iota
	// MessageNotEpisode did not parse as an episode
	MessageNotEpisode
	// MessageNotTracked did not contain a tracked episode
	MessageNotTracked
	// MessageIncorrectQuality had the wrong video quality
	MessageIncorrectQuality
	// MessageNotNewEpisode did not contain a new episode
	MessageNotNewEpisode
	// MessageNoURL had no parsable URL
	MessageNoURL
	// MessageFetchURL passed all tests and should be fetched
	MessageFetchURL
)

// Matcher is a grouping of regexps used in validating strings that are passed here.
type Matcher struct {
	State config.ProcessState

	announceRegexp *regexp.Regexp
	episodeRegexp  *regexp.Regexp
	qualityRegexp  *regexp.Regexp

	f  fetcher.Fetch
	l  *log.Logger
	tc *config.TrackerConfig

	mutex sync.RWMutex
}

// New creates a *matcher.Matcher pointer to annalyze the data delivered by watchers.
func New(tc *config.TrackerConfig, l *log.Logger, g fetcher.Fetch) *Matcher {
	return &Matcher{
		f:  g,
		l:  l,
		tc: tc,
	}
}

// CheckMessage is pretty explanitory
func (m *Matcher) CheckMessage(message string) MatchState {
	var l *url.URL
	var err error

	mapPattern := m.matchEpisodeToPattern(message)
	if mapPattern == nil {
		return l, MessageNotEpisode
	}
	sid := m.isTorrentAndTracked(mapPattern["show"])
	if sid == nil {
		return l, MessageNotTracked
	}
	if mapPattern["quality"] != sid.Quality {
		return l, MessageIncorrectQuality
	}

	if !m.isNewEpisode(sid.ID, mapPattern) {
		return l, MessageNotNewEpisode
	}
	if l, err = url.Parse(mapPattern["url"]); err != nil {
		return l, MessageNoURL
	}
	return l, MessageFetchURL
}

// Update is blah blah blah
func (m *Matcher) Update(tc *config.TrackerConfig) error {
	m.l.Println("Updating matcher configs")
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var err error
	if m.announceRegexp, err = regexp.Compile(tc.IRC.AnnounceRegexp); err != nil {
		return err
	}
	if m.episodeRegexp, err = regexp.Compile(tc.IRC.EpisodeRegexp); err != nil {
		return err
	}
	if m.qualityRegexp, err = regexp.Compile(tc.IRC.QualityRegexp); err != nil {
		return err
	}
	return nil
}

func (m *Matcher) matchEpisodeToPattern(e string) map[string]string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	match := m.episodeRegexp.FindAllStringSubmatch(e, -1)
	if match == nil {
		m.l.Println("Not episode announcement")
		return nil
	}

	var named map[string]string
	for i, n := range match[0] {
		named[m.episodeRegexp.SubexpNames()[i]] = n
	}
	m.l.Println("Episode announcement found")
	return named
}

func (m *Matcher) isTorrentAndTracked(show string) *db.Show {
	rewritenShow := misc.EpisodeRewriter(show)
	sid, err := db.GetShowByTitle(rewritenShow)
	if err != nil {
		m.l.Println("Untracked show found: ", rewritenShow)
		return nil
	}
	return sid
}

func (m *Matcher) isQuality(newQ, expQ string) error {
	if newQ != expQ {
		return fmt.Errorf("No quality match")
	}
	return nil
}

func (m *Matcher) isNewEpisode(sid int64, mp map[string]string) bool {
	s, _ := strconv.Atoi(mp["season"])
	e, _ := strconv.Atoi(mp["episode"])
	return db.NewEpisode(sid, s, e).IsNewEpisode()
}
