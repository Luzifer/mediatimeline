function sortOrder(i, j) {
  switch (true) {
  case i < j:
    return -1
  case j < i:
    return 1
  default:
    return 0
  }
}

new Vue({
  computed: {
  },

  data: {
    tweets: [],
    modalTweet: null,
  },

  el: '#app',

  methods: {
    callModal(tweet) {
      this.modalTweet = tweet
    },

    deleteTweet(tweet) {
      axios
        .delete(`/api/${tweet.id}`)
        .then(() => this.removeTweet(tweet.id))
        .catch(err => console.log(err))
    },

    favourite(tweet) {
      axios
        .put(`/api/${tweet.id}/favorite`)
        .then(res => {
          if (res.data.length === 0) {
            this.refetch(tweet)
            return
          }

          this.upsertTweets(res.data)
        })
        .catch(err => console.log(err))
    },

    notify(text, title = 'MediaTimeline Viewer') {
      this.$bvToast.toast(text, {
        title,
        autoHideDelay: 3000,
        appendToast: true,
      })
    },

    refetch(tweet) {
      axios
        .put(`/api/${tweet.id}/refresh`)
        .then(res => {
          if (res.data.gone) {
            return this.removeTweet(tweet.id)
          }

          if (res.data.length === 0) {
            return
          }

          this.upsertTweets(res.data)
        })
        .catch(err => console.log(err))
    },

    refresh(forceReload = false) {
      let apiURL = '/api/page' // By default query page 1
      let append = false
      if (this.tweets.length > 0 && !forceReload) {
        apiURL = `/api/since?id=${this.tweets[0].id}`
        append = true
      }

      axios
        .get(apiURL)
        .then(resp => {
          if (resp.data.length === 0) {
            return
          }

          this.upsertTweets(resp.data, append)
        })
        .catch(err => {
          console.log(err)
        })
    },

    removeTweet(id) {
      const tweets = []

      for (const i in this.tweets) {
        const t = this.tweets[i]
        if (t.id === id) {
          continue
        }

        tweets.push(t)
      }

      this.tweets = tweets
    },

    triggerForceFetch() {
      axios
        .put('/api/force-reload')
        .then(() => {
          this.notify('Force refresh triggered, reloading tweets in 10s')
          window.setTimeout(() => this.refresh(true), 10000)
        })
        .catch(err => console.log(err))
    },

    upsertTweets(data, append = true) {
      let tweets = append ? this.tweets : []

      for (const idx in data) {
        const tweet = data[idx]
        let inserted = false

        for (let i = 0; i < tweets.length; i++) {
          if (tweets[i].id === tweet.id) {
            tweets[i] = tweet
            inserted = true
            break
          }
        }

        if (!inserted) {
          tweets = [...tweets, tweet]
        }
      }

      tweets.sort((j, i) => sortOrder(i.id, j.id))
      this.tweets = tweets
    },
  },

  mounted() {
    this.refresh()
    window.setInterval(this.refresh, 30000)
  },

})
