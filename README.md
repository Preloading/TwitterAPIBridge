# A Twitter Bridge
## from bluesky to twitter api v1
###### This is not affiliated with Twitter (now X) and Bluesky.

![An iPhone 3G, iPhone 4S, and Nexus 4 showing the Twitter home timeline](https://raw.githubusercontent.com/Preloading/TwitterAPIBridge/refs/heads/main/resources/1.png)

This custom server translates Twitter V1 requests to Bluesky for Twitter clients.

# Compatibility
To see what devices and versions are compatible, look at [the compatibility list](https://github.com/Preloading/TwitterAPIBridge/blob/main/COMPATIBILITY.md)

# Public Instances

https://twitterbridge.loganserver.net HTTP & HTTPS, Based on Releases (my instance)

https://testtwitterbridge.loganserver.net HTTP & HTTPS, Based on commits (my instance) 

## Usage
### iOS
1. Install the IPA of your choosing (you can find some here, along with android versions: https://loganserver.net/twitters/, also elon don't sue me pls). At present moment, recommended is 4.1.3, latest to work is 5.0.3 for offical iOS)
2. Create a bluesky app password (assuming you have a bluesky account), menu > settings > privacy and security > app passwords. You will want to enable DMs for future DM compatibility.
3. Open the Twitter app
4. Click the login button
5. Click the cog button
6. Put your instance for both urls (you can make it http if on ios 3 provided the server supports it)
7. Type in your bluesky handle for the username, and your bluesky app password as the password. **Usage of normal passwords is not recommended, and will be forbidden in the future**
8. Login
9. Hopefully success


## Hosting instructions
For hosting, you can take two paths
- 🐳 Docker - Quick & easy to deploy + easy updating
- 🖥 Bare metal (recommended) - Useful when developing & debugging


### 🐳 Docker
Warning: Docker at present moment has some *preformance issues*, if anyone competent in docker has any idea why, please leave a github issue :)

I assume you have some competency in docker.

Docker Compose:
```yaml
services:
  twitter-bridge:
    # image: ghcr.io/preloading/twitterapibridge:main # main = releases/stable
    image: ghcr.io/preloading/twitterapibridge:dev # dev=latest commits/test version
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
      - '/opt/twitterbridge/sqlite:/config/sqlite' # Path where the SQLite DB is stored. It is safe to remove if you aren't using SQLite for your database.
```

|Env|Function|Default|
| :----: | --- | :---: |
|``TWITTER_BRIDGE_DATABASE_TYPE``| They type of database to connect to. Options include sqlite, mysql, and postgres. |``"sqlite"``|
|``TWITTER_BRIDGE_DATABASE_PATH``| Changes where it looks for the database (Path/DSN) (see https://gorm.io/docs/connecting_to_the_database.html) |``"/config/database/sqlite.db"``|
|``TWITTER_BRIDGE_CDN_URL``| The CDN_URL is the URL where clients can access images from this server. Do not include a trailing slash. | ``"http://127.0.0.1:3000"`` |
|``TWITTER_BRIDGE_SERVER_PORT``| The port where the server is running |``3000``|
|``TWITTER_BRIDGE_TRACK_ANALYTICS``| Enables tracking of analytics (at the moment the only way to view this is by looking at the database) |``true``|
|``TWITTER_BRIDGE_DEVELOPER_MODE``| Enables extra loggging of data useful for debugging. WARNING!: DO NOT ENABLE ON A PUBLIC INSTANCE!!!! |``false``|

### 🖥 Bare metal (recommended)
This assumes you are somewhat competent
#### 1. Clone the repo
```bash
git clone https://github.com/Preloading/TwitterAPIBridge.git
```
#### 2. Duplicate and rename config.sample.yaml as config.yaml
#### 3. Configure config.yaml (edit the file, you'll see decription of what each thing does)
#### 4. Run/build
If you want to just run this in the directory when you found config.sample.yaml
```
go run .
```
If you want to build the project run this in the directory when you found config.sample.yaml
```
go build .
```
#### 5. (if building) open the executable
#### 6. (hopefully) success!

## Accuracy
This server is no where close to 100% accurate. Most of the the time this accuracy is having more values responed than what should be, and it shouldn't affect clients using this.

## Thanks to
[@Preloading](https://github.com/Preloading), I wrote the thing

[@Savefade](https://github.com/Savefade), Gave me info on some of the requests

[@retrofoxxo](https://github.com/retrofoxxo), Helped with getting android working with this server
