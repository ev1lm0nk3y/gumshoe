// Project matcher is your onestop shop for all log message processing and validation.
package matcher

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"

	"github.com/ev1lm0nk3y/gumshoe/db"
	"github.com/ev1lm0nk3y/gumshoe/misc"
)

type matcherOps int

const (
	matcherOpsList matcherOps = iota
	Accept
	Ignore
	Error
)

// Matcher is a grouping of regexp for the cloudley project.
type Matcher struct {
	AnnounceRegexp *regexp.Regexp
	EpisodeRegexp  *regexp.Regexp
	QualityRegexp  *regexp.Regexp

	AnnChan chan string
	OutChan chan matcherOps
	ErrChan chan error

	Link chan *url.URL

	logger *log.Logger
}

// New creates a *matcher.Matcher pointer to annalyze the data returned from the watchers.
func New(announce, episode, quality string, logger *log.Logger) *Matcher {
	return &Matcher{
		AnnounceRegexp: regexp.MustCompile(announce),
		EpisodeRegexp:  regexp.MustCompile(episode),
		QualityRegexp:  regexp.MustCompile(quality),

		AnnChan: make(chan string, 10),
		OutChan: make(chan matcherOps),
		ErrChan: make(chan error),

		logger: logger,
	}
}

// Run takes inputs from a channel and tests the input to see if it can proceed
// to fetching.
func (m *Matcher) Run() error {
	m.logger.Println("Running matcher service")
	for {
		a := <-m.aChan
		go m.generateResponse(a)
	}
}

func (m *Matcher) generateResponse(message string) {
	mapPattern, err := m.matchEpisodeToPattern(message)
	if err != nil {
		m.ErrChan <- err
		m.OutChan <- Error
	}
	sid, err := m.isTorrentAndTracked(message)
	if err != nil {
		m.ErrChan <- err
		m.OutChan <- Error
	}
	if err := m.isQuality(mapPattern["quality"], sid.Quality); err != nil {
		m.ErrChan <- err
		m.OutChan <- Ignore
	}
	season, _ := strconv.Atoi(mapPattern["season"])
	episode, _ := strconv.Atoi(mapPattern["episode"])
	if m.isNewEpisode(sid.ID, season, episode) {
		l, err := url.Parse(mapPattern["url"])
		if err != nil {
			m.ErrChan <- fmt.Errorf("[ERROR] %v", err)
			continue
		}
		m.OutChan <- Accept
		m.Link <- l
	}
	m.OutChan <- Ignore
	m.ErrChan <- nil
}

func (m *Matcher) matchEpisodeToPattern(e string) (map[string]string, error) {
	var named map[string]string
	match := m.EpisodeRegexp.FindAllStringSubmatch(e, -1)
	if match == nil {
		return nil, fmt.Errorf("string %s not matched episode regexp", e)
	}

	for i, n := range match[0] {
		named[m.EpisodeRegexp.SubexpNames()[i]] = n
	}
	return named, nil
}

func (m *Matcher) isTorrentAndTracked(e string) (*db.Show, error) {
	eMatch, err := m.matchEpisodeToPattern(e)
	if err != nil {
		return nil, err
	}
	rewritenShow := misc.EpisodeRewriter(eMatch["show"])
	sid, err := db.GetShowByTitle(rewritenShow)
	if err != nil {
		return nil, fmt.Errorf("Show %s is not being tracked", rewritenShow)
	}
	return sid, nil
}

func (m *Matcher) isQuality(newQ, expQ string) error {
	if newQ != expQ {
		return fmt.Errorf("No quality match")
	}
	return nil
}

func (m *Matcher) isNewEpisode(sid int64, s, e int) bool {
	return db.NewEpisode(sid, s, e).IsNewEpisode()
}
