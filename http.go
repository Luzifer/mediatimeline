package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ChimeraCoder/anaconda"
	log "github.com/sirupsen/logrus"
)

func init() {
	http.HandleFunc("/api/page", handlePage)
	http.HandleFunc("/api/since", handleNewest)
	http.HandleFunc("/api/favourite", handleFavourite)
	http.HandleFunc("/api/refresh", handleTweetRefresh)
	http.HandleFunc("/api/force-reload", handleForceReload)
}

func handlePage(w http.ResponseWriter, r *http.Request) {
	var page int = 1

	if p, err := strconv.Atoi(r.URL.Query().Get("n")); err == nil && p > 0 {
		page = p
	}

	tweets, err := tweetStore.GetTweetPage(page)
	if err != nil {
		log.WithError(err).Error("Unable to fetch tweets for page request")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tweetResponse(w, tweets)
}

func handleNewest(w http.ResponseWriter, r *http.Request) {
	var since uint64 = 0

	if s, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64); err == nil && s > 0 {
		since = s
	}

	if since == 0 {
		http.Error(w, "Must specify last id", http.StatusBadRequest)
		return
	}

	tweets, err := tweetStore.GetTweetsSince(since)
	if err != nil {
		log.WithError(err).Error("Unable to fetch tweets for newest request")
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tweetResponse(w, tweets)
}

func tweetResponse(w http.ResponseWriter, tweets []tweet) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(tweets)
}

func handleFavourite(w http.ResponseWriter, r *http.Request) {
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

func handleForceReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "This needs to be POST", http.StatusBadRequest)
		return
	}

	go loadAndStoreTweets(true)

	w.WriteHeader(http.StatusNoContent)
}
