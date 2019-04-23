package main

import (
	"encoding/gob"
	"os"
	"sort"
	"sync"

	"github.com/pkg/errors"
)

const tweetPageSize = 50

type store struct {
	s        []tweet
	location string

	lock sync.RWMutex
}

func init() {
	gob.Register(tweet{})
}

func newStore(location string) (*store, error) {
	s := &store{
		s:        []tweet{},
		location: location,
	}

	return s, s.load()
}

func (s *store) StoreTweets(tweets []tweet) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	tmp := s.s

	for _, t := range tweets {
		var stored bool

		for i := 0; i < len(tmp); i++ {
			if tmp[i].ID == t.ID {
				tmp[i] = t
				stored = true
				break
			}
		}

		if !stored {
			tmp = append(tmp, t)
		}
	}

	sort.Slice(tmp, func(j, i int) bool { return tmp[i].ID < tmp[j].ID })

	s.s = tmp

	return s.save()
}

func (s *store) GetLastTweetID() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(s.s) == 0 {
		return 0
	}

	return s.s[0].ID
}

func (s *store) GetTweetPage(page int) ([]tweet, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var (
		start = (page - 1) * tweetPageSize
		num   = tweetPageSize
	)

	if start > len(s.s) {
		return []tweet{}, nil
	}

	if start+num >= len(s.s) {
		num = len(s.s) - start
	}

	return s.s[start:num], nil
}

func (s *store) GetTweetsSince(since uint64) ([]tweet, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var i int

	for i = 0; i < len(s.s); i++ {
		if s.s[i].ID <= since {
			break
		}
	}

	return s.s[:i], nil
}

func (s *store) load() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, err := os.Stat(s.location); err != nil {
		if os.IsNotExist(err) {
			// Leave a fresh store
			return nil
		}

		return errors.Wrap(err, "Unable to stat storage file")
	}

	f, err := os.Open(s.location)
	if err != nil {
		return errors.Wrap(err, "Unable to open storage file")
	}
	defer f.Close()

	tmp := []tweet{}

	if err := gob.NewDecoder(f).Decode(&tmp); err != nil {
		return errors.Wrap(err, "Unable to decode storage file")
	}

	s.s = tmp

	return nil
}

func (s *store) save() error {
	// No need to lock here, has write-lock from s.StoreTweets

	f, err := os.Create(s.location)
	if err != nil {
		return errors.Wrap(err, "Unable to open store for writing")
	}
	defer f.Close()

	return errors.Wrap(gob.NewEncoder(f).Encode(s.s), "Unable to encode store")
}
