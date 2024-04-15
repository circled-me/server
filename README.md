# circled.me community server
This project aims to help people easily backup and share photos, videos, albums on their own server. Focusing on performance, low footprint and ease of implementation and use.
The application is not dependant on any other service if you use the default SQLite DB engine. Currently SQLite and MySQL are supported for metadata storage.

After certain services that scan for faces, locations, etc, became paid some time ago, I have decided I'd rather be able to host my own photos.
The main reason being, of course, privacy! But also at that time, there was no alternatve that offered good performance and low CPU/memory usage. 

Another focus of this project is having the ability to host everything a community needs to be able to communicate and exchange photos, ideas, etc.
I strongly believe in local/focused communities and sharing with the community, but at the same time - keeping everything private, within the community.
In my personal case, I share mostly photos with my family and close friends.

Logo is <a href="http://madebytow.com/">madebytow.com</a>

## Mobile app
The **circled.me** mobile app **works with multiple accounts and servers**. For example, you can have your family server and account, and your gaming/running/reading comunities' accounts on the same app and being able to interact with all of them at the same time. It is currently the only way to interface with the server. Go to https://circled.me to download it.

<img src="https://app.circled.me/screenshots.jpg"/>

___

⚠️ **NOTE: Please note that this project is still in development and could introduce breaking changes.**

⚠️ **WARNING: Do not use this as your main/only backup solution.**

___


## Main features:
- Fast response times and low CPU and memory usage
- iOS and Android photo backup (using the circled.me app available on the AppStore and Google Play)
  - Supports either locally mounted disks or
  - S3-compatible Services - this allows different users to use their own S3 bucket on the same server
- Push notifications for new Album photos, etc
- Albums
  - Adding local server contributors and viewers
  - Sharing albums with anyone with a "secret" link
- Chat over websockets and with push notifications
- Filtering photos by year, month, location, etc
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
- `MYSQL_DSN` - see example or refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
- `BIND_ADDRESS` - IP and port to bind to (incompatible with `TLS_DOMAINS`). This is useful if your server is, say, behind reverse proxy
- `TLS_DOMAINS` - a list of comma-separated domain names. This uses the Let's Encrypt Gin implementation (https://github.com/gin-gonic/autotls)
- `DEBUG_MODE` - currently defaults to `yes`

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
