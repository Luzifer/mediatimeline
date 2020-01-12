package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ChimeraCoder/anaconda"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func init() {
	router.HandleFunc("/api/{tweetID}/favorite", handleFavorite).Methods(http.MethodPut)
	router.HandleFunc("/api/{tweetID}", handleDelete).Methods(http.MethodDelete)
	router.HandleFunc("/api/force-reload", handleForceReload).Methods(http.MethodPut)
	router.HandleFunc("/api/page", handlePage).Methods(http.MethodGet)
	router.HandleFunc("/api/{tweetID}/refresh", handleTweetRefresh).Methods(http.MethodPut)
	router.HandleFunc("/api/since", handleNewest).Methods(http.MethodGet)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	var vars = mux.Vars(r)

	tweetID, err := strconv.ParseInt(vars["tweetID"], 10, 64)
	if err != nil {
		http.Error(w, errors.Wrap(err, "Unable to parse TweetID").Error(), http.StatusBadRequest)
		return
	}

	if err := tweetStore.DeleteTweetByID(uint64(tweetID)); err != nil {
		log.WithError(err).Error("Unable to delete tweet")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleFavorite takes the ID of a tweet and submits a favorite to Twitter
func handleFavorite(w http.ResponseWriter, r *http.Request) {
	var vars = mux.Vars(r)

	tweetID, err := strconv.ParseInt(vars["tweetID"], 10, 64)
	if err != nil {
		http.Error(w, errors.Wrap(err, "Unable to parse TweetID").Error(), http.StatusBadRequest)
		return
	}

	tweet, err := twitter.Favorite(tweetID)
	if err != nil {
		log.WithError(err).Error("Unable to favourite tweet")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tweets, err := convertTweetList([]anaconda.Tweet{tweet}, true)
	if err != nil {
		log.WithError(err).Error("Unable to convert tweet for storing")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err = tweetStore.StoreTweets(tweets); err != nil {
		log.WithError(err).Error("Unable to update tweet")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tweetResponse(w, tweets)
}

// handleForceReload issues a full load of the latest tweets to update their state
func handleForceReload(w http.ResponseWriter, r *http.Request) {
	go loadAndStoreTweets(true)

	w.WriteHeader(http.StatusNoContent)
}

// handleNewest returns all tweets newly stored since the given tweet ID
func handleNewest(w http.ResponseWriter, r *http.Request) {
	var since uint64

	if s, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64); err == nil && s > 0 {
		since = s
	}

	if since == 0 {
		http.Error(w, "Must specify last id", http.StatusBadRequest)
		return
	}

	tweetResponse(w, tweetStore.GetTweetsSince(since))
}

// handlePage loads older tweets with pagination
func handlePage(w http.ResponseWriter, r *http.Request) {
	var page = 1

	if p, err := strconv.Atoi(r.URL.Query().Get("n")); err == nil && p > 0 {
		page = p
	}

	tweetResponse(w, tweetStore.GetTweetPage(page))
}

// handleTweetRefresh refreshes the state of the tweet with the given ID against the Twitter API
func handleTweetRefresh(w http.ResponseWriter, r *http.Request) {
	var vars = mux.Vars(r)

	tweetID, err := strconv.ParseInt(vars["tweetID"], 10, 64)
	if err != nil {
		http.Error(w, errors.Wrap(err, "Unable to parse TweetID").Error(), http.StatusBadRequest)
		return
	}

	tweet, err := twitter.GetTweet(tweetID, url.Values{})
	if err != nil {
		if strings.Contains(err.Error(), "No status found with that ID.") {
			log.WithField("id", tweetID).Info("Removing no longer existing tweet")
			if err = tweetStore.DeleteTweetByID(uint64(tweetID)); err != nil {
				log.WithError(err).Error("Unable to delete tweet")
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			jsonResponse(w, map[string]bool{"gone": true})
			return
		}

		log.WithError(err).Error("Unable to fetch tweet")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tweets, err := convertTweetList([]anaconda.Tweet{tweet}, true)
	if err != nil {
		log.WithError(err).Error("Unable to convert tweet for storing")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err = tweetStore.StoreTweets(tweets); err != nil {
		log.WithError(err).Error("Unable to update tweet")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tweetResponse(w, tweets)
}

// tweetResponse is a generic wrapper to return a list of tweets through JSON
func tweetResponse(w http.ResponseWriter, tweets []tweet) {
	jsonResponse(w, tweets)
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(data)
}
