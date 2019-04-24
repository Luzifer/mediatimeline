package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ChimeraCoder/anaconda"
	log "github.com/sirupsen/logrus"
)

func init() {
	http.HandleFunc("/api/favourite", handleFavorite)
	http.HandleFunc("/api/force-reload", handleForceReload)
	http.HandleFunc("/api/page", handlePage)
	http.HandleFunc("/api/refresh", handleTweetRefresh)
	http.HandleFunc("/api/since", handleNewest)
}

// handleFavorite takes the ID of a tweet and submits a favorite to Twitter
func handleFavorite(w http.ResponseWriter, r *http.Request) {
	req := struct {
		ID int64 `json:"id,string"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		http.Error(w, "Need to specify id", http.StatusBadRequest)
		return
	}

	tweet, err := twitter.Favorite(req.ID)
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
	if r.Method != http.MethodPost {
		http.Error(w, "This needs to be POST", http.StatusBadRequest)
		return
	}

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
	req := struct {
		ID int64 `json:"id,string"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == 0 {
		http.Error(w, "Need to specify id", http.StatusBadRequest)
		return
	}

	tweet, err := twitter.GetTweet(req.ID, url.Values{})
	if err != nil {
		if strings.Contains(err.Error(), "No status found with that ID.") {
			log.WithField("id", req.ID).Info("Removing no longer existing tweet")
			if err = tweetStore.DeleteTweetByID(uint64(req.ID)); err != nil {
				log.WithError(err).Error("Unable to delete tweet")
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
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
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(tweets)
}
