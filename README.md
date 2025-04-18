# Circled.me Community Server
This project aims to help people easily backup and share photos, videos, albums on their own server and to enable communication, including audio/video calls and chats. And do all this by keeping everything private. Focusing on performance, low footprint and ease of implementation and use. The application is not dependant on any other service if you use the default SQLite DB engine.

Having the ability to host everything a community needs to be able to communicate and exchange photos, ideas, etc, is the main focus here.
I strongly believe in local/focused communities and sharing with the community, but at the same time - keeping everything private, within the community.
In my personal case, I share mostly photos with my family and close friends, and also use the video call functionality to talk to them.

Logo is <a href="http://madebytow.com/">madebytow.com</a>

## Mobile app
The **circled.me** mobile app **works with multiple accounts and servers**. For example, you can have your family server and account, and your gaming/running/reading comunities' accounts on the same app and being able to interact with all of them at the same time. It is currently the only way to interface with the server. Go to https://circled.me to download it or go to the source repo here: https://github.com/circled-me/app

<img src="https://app.circled.me/screenshots.jpg"/>

___

⚠️ **NOTE: Please note that this project is still in development and could introduce breaking changes.**

⚠️ **WARNING: Do not use this as your main/only backup solution.**

___


## Main features:
- Fast response times and low CPU and very low memory footprint
- iOS and Android photo backup (using the circled.me app available on the AppStore and Google Play)
  - Supports either locally mounted disks or
  - S3-compatible Services - this allows different users to use their own S3 bucket on the same server
- Push notifications for new Album photos, etc
- Video/Audio Calls using the mobile app OR any browser
- Face detection and tagging
- Albums
  - Adding local server contributors and viewers
  - Sharing albums with anyone with a "secret" link
- Chat with push notifications
- Filtering photos by tagged person, year, month, location, etc
- Moments - automatically grouping photos by time and location
- Reverse geocoding for all assets
- Automatic video conversion to web-compatible H.264 format


## Compiling and Running the server
To compile the server you will need go 1.21 or above and simply build it within the cloned repository: `CGO_ENABLED=1 go build`.

The easiest way to try and run the server is to use the latest image available on Docker Hub, see example docker-compose file below.
The latest version of the server uses SQLite as default DB engine.

```bash
docker-compose -f docker-compose-example.yaml up
```

Now you can use the app and connect to your server at `http://<YOUR_IP>:8080` and create your first (admin) user.

Current configuration environment variables:
- `SQLITE_FILE` - location of the SQLite file. If non-existent, a new DB file will be created with the given name. Note that MySQL below takes precedence (if both configured)
- `MYSQL_DSN` - see example or refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
- `BIND_ADDRESS` - IP and port to bind to (incompatible with `TLS_DOMAINS`). This is useful if your server is, say, behind reverse proxy
- `TLS_DOMAINS` - a list of comma-separated domain names. This uses the Let's Encrypt Gin implementation (https://github.com/gin-gonic/autotls)
- `DEBUG_MODE` - currently defaults to `yes`
- `DEFAULT_BUCKET_DIR` - a directory that will be used as default bucket if no other buckets exist (i.e. the first time you run the server)
- `DEFAULT_ASSET_PATH_PATTERN` - the default path pattern to create subdirectories and file names based on asset info. Defaults to `<year>/<month>/<id>`
- `PUSH_SERVER` - the push server URL. Defaults to `https://push.circled.me`
- `FACE_DETECT` - enable/disable face detection. Defaults to `yes`
- `FACE_DETECT_CNN` - use Convolutional Neural Network for face detection (as opposed to HOG). Much slower, but more accurate at different angles. Defaults to `no`
- `FACE_MAX_DISTANCE_SQ` - squared distance between faces to consider them similar. Defaults to `0.11`
- `TURN_SERVER_IP` - if configured, Pion TURN server would be started locally and this value used to advertise ourselves. Should be your public IP. Defaults to empty string
- `TURN_SERVER_PORT` - Defaults to port '3478' (UDP)
- `TURN_TRAFFIC_MIN_PORT` and `TURN_TRAFFIC_MAX_PORT` - Advertise-able UDP port range for TURN traffic. Those ports need to be open on your public IP (and forwarded to the circled.me server instance). Defaults to 49152-65535

## docker-compose example
```yaml
version: '2'
services:
  circled-server:
    image: gubble/circled-server:latest
    restart: always
    ports:
      - "8080:8080"
    environment:
      SQLITE_FILE: "/mnt/data1/circled.db"
      BIND_ADDRESS: "0.0.0.0:8080"
      DEFAULT_BUCKET_DIR: "/mnt/data1"
      DEFAULT_ASSET_PATH_PATTERN: "<year>/<month>/<id>"
    volumes:
      - ./circled-data:/mnt/data1
```
The project includes an example docker-compose file with MySQL configuration.
