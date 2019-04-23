package main

import (
	"strconv"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/pkg/errors"
)

func convertTweetList(in []anaconda.Tweet, filterForMedia bool) ([]tweet, error) {
	out := []tweet{}

	for _, t := range in {
		tw, err := tweetFromAnaconda(t)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to parse tweet")
		}

		if len(tw.Images) == 0 && filterForMedia {
			continue
		}
		out = append(out, tw)
	}

	return out, nil
}

type tweet struct {
	Favorited bool      `json:"favorited"`
	ID        uint64    `json:"id,string"`
	Images    []media   `json:"images"`
	Posted    time.Time `json:"posted"`
	User      user      `json:"user"`
	Text      string    `json:"text"`
}

func tweetFromAnaconda(in anaconda.Tweet) (tweet, error) {
	medias := []media{}
	for _, m := range in.ExtendedEntities.Media {
		if m.Type != "photo" {
			continue
		}

		medias = append(medias, mediaFromAnaconda(m))
	}

	created, _ := in.CreatedAtTime()
	id, err := strconv.ParseUint(in.IdStr, 10, 64)
	if err != nil {
		return tweet{}, errors.Wrap(err, "Unable to parse tweet ID")
	}

	return tweet{
		Favorited: in.Favorited,
		ID:        id,
		Images:    medias,
		Posted:    created,
		User:      userFromAnaconda(in.User),
		Text:      in.Text,
	}, nil
}

type user struct {
	ID         int64  `json:"id"`
	ScreenName string `json:"screen_name"`
	Image      string `json:"image"`
}

func userFromAnaconda(in anaconda.User) user {
	return user{
		ID:         in.Id,
		ScreenName: in.ScreenName,
		Image:      in.ProfileImageUrlHttps,
	}
}

type media struct {
	ID    int64  `json:"id"`
	Image string `json:"image"`
}

func mediaFromAnaconda(in anaconda.EntityMedia) media {
	return media{
		ID:    in.Id,
		Image: in.Media_url_https,
	}
}
