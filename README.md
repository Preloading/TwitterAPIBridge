# Bluesky -> Twitter V1 Bridge

This custom server translates Twitter V1 requests to Bluesky for Twitter clients.

The primary target for this is the iPhone 3G with Twitter 4.1.3

I do not recommend hosting a public instance of this software yet, as work is still being done, and credentials may be visible to the hoster.

I will not be guarenting anything. I'll try to target both bluesky & mastodon, but no guarentee yet. Right now looking like bluesky will be the primary target due to some technical challenges with mastodon

## Hosting instructions

Docker
```yaml
services:
  twitter-bridge:
    image: ghcr.io/preloading/twitterapibridge:dev # main = releases/stable, dev=latest commits/test version
    environment:
      - TWITTER_BRIDGE_DATABASE_TYPE=sqlite
      - TWITTER_BRIDGE_DATABASE_PATH=/config/database/sqlite.db
      - TWITTER_BRIDGE_CDN_URL=http://127.0.0.1
      - TWITTER_BRIDGE_SERVER_PORT=3000
      - TWITTER_BRIDGE_TRACK_ANALYTICS=true
      - TWITTER_BRIDGE_DEVELOPER_MODE=false
    ports:
      - "80:3000"
    volumes:
      - '/opt/testtwitterbridge/sqlite:/config/sqlite'
```

### This is not (and probably won't be for a while) a 100% accurate recreation of the twitter API

most of this readme todo

## Thanks
@Preloading - I wrote the thing
@Savefade - Gave me info on some of the requests