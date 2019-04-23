package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ChimeraCoder/anaconda"
	log "github.com/sirupsen/logrus"

	"github.com/Luzifer/rconfig/v2"
)

var (
	cfg = struct {
		AppToken       string `flag:"app-token" description:"Token for the application" validate:"nonzero"`
		AppSecret      string `flag:"app-secret" description:"Secret for the provided token" validate:"nonzero"`
		Database       string `flag:"database" default:"tweets.db" description:"Database storage location"`
		Frontend       string `flag:"frontend" default:"frontend" description:"Directory containing frontend files"`
		Listen         string `flag:"listen" default:":3000" description:"Port/IP to listen on"`
		ListOwner      string `flag:"list-owner" description:"Owner of the specified list" validate:"nonzero"`
		ListSlug       string `flag:"list-slug" description:"Slug of the list" validate:"nonzero"`
		LogLevel       string `flag:"log-level" default:"info" description:"Log level (debug, info, warn, error, fatal)"`
		UserToken      string `flag:"user-token" description:"Token for the user" validate:"nonzero"`
		UserSecret     string `flag:"user-secret" description:"Secret for the provided token" validate:"nonzero"`
		VersionAndExit bool   `flag:"version" default:"false" description:"Prints current version and exits"`
	}{}

	tweetStore *store
	twitter    *anaconda.TwitterApi

	version = "dev"
)

func init() {
	rconfig.AutoEnv(true)
	if err := rconfig.ParseAndValidate(&cfg); err != nil {
		log.Fatalf("Unable to parse commandline options: %s", err)
	}

	if cfg.VersionAndExit {
		fmt.Printf("mediatimeline %s\n", version)
		os.Exit(0)
	}

	if l, err := log.ParseLevel(cfg.LogLevel); err != nil {
		log.WithError(err).Fatal("Unable to parse log level")
	} else {
		log.SetLevel(l)
	}
}

func main() {
	var err error
	if tweetStore, err = newStore(cfg.Database); err != nil {
		log.WithError(err).Fatal("Unable to create store")
	}

	twitter = anaconda.NewTwitterApiWithCredentials(
		cfg.UserToken, cfg.UserSecret,
		cfg.AppToken, cfg.AppSecret,
	)
	go func() {
		for t := time.NewTicker(time.Minute); true; <-t.C {
			loadAndStoreTweets(false)
		}
	}()

	http.Handle("/", http.FileServer(http.Dir(cfg.Frontend)))
	http.ListenAndServe(cfg.Listen, nil)
}

func loadAndStoreTweets(forceRefresh bool) {
	params := url.Values{
		"count": []string{"100"},
	}

	lastTweet := tweetStore.GetLastTweetID()

	if lastTweet > 0 && !forceRefresh {
		params.Set("since_id", strconv.FormatUint(lastTweet, 10))
	}

	anacondaTweets, err := twitter.GetListTweetsBySlug(cfg.ListSlug, cfg.ListOwner, false, params)
	if err != nil {
		log.WithError(err).Error("Unable to fetch tweets")
		return
	}

	tweets, err := convertTweetList(anacondaTweets, true)
	if err != nil {
		log.WithError(err).Error("Unable to parse tweets")
		return
	}

	if err := tweetStore.StoreTweets(tweets); err != nil {
		log.WithError(err).Error("Unable to store tweets")
		return
	}
}
