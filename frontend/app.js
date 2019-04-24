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

    favourite(tweet) {
      axios
        .post('/api/favourite', { id: tweet.id })
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
        .post('/api/refresh', { id: tweet.id })
        .then(res => {
          if (res.data.length === 0) {
            return
          }

          this.upsertTweets(res.data)
        })
        .catch(err => console.log(err))
    },

    refresh(forceReload = false) {
      let apiURL = '/api/page' // By default query page 1
      if (this.tweets.length > 0 && !forceReload) {
        apiURL = `/api/since?id=${this.tweets[0].id}`
      }

      axios
        .get(apiURL)
        .then(resp => {
          if (resp.data.length === 0) {
            return
          }

          this.upsertTweets(resp.data)
        })
        .catch(err => {
          console.log(err)
        })
    },

    triggerForceFetch() {
      axios
        .post('/api/force-reload')
        .then(() => {
          this.notify('Force refresh triggered, reloading tweets in 10s')
          window.setTimeout(() => this.refresh(true), 10000)
        })
        .catch(err => console.log(err))
    },

    upsertTweets(data) {
      let tweets = this.tweets

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
