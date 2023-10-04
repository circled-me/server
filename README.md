# circled.me community server
This project aims to help people easily backup and share photos, videos, albums on their own server. Focusing on performance, low footprint and ease of implementation and use.
Upcoming releases will further enable you to share with your circles by including group chats and more.

After certain services that scan for faces, locations, etc, became paid some time ago, I have decided I'd rather be able to host my own photos.
The main reason being, of course, privacy! But also at that time, there was no alternatve that offered good performance and low CPU/memory usage. 
This project has currently only one contributor (i.e. me), so help will be greatly appreciated 😊

Another focus of this project is having the ability to host everything a community needs to be able to communicate and exchange photos, ideas, etc.
I strongly believe in local/focused communities and sharing with the community, but at the same time - keeping everything private, within the community.
In my personal case, I share mostly photos with my family and close friends.

Logo is <a href="http://madebytow.com/">madebytow.com</a>

## Mobile app
The **circled.me** mobile app **works with multiple accounts and servers**. For example, you can have your family server and account, and your gaming/running/reading comunities' accounts on the same app and being able to interact with all of them at the same time.

<img src="https://app.circled.me/screenshots.jpg"/>

___

⚠️ **NOTE: Please note that this project is still in development and could introduce breaking changes.**

⚠️ **WARNING: Do not use this as your main/only backup solution.**

___


## Main features:
- Fast response times and low CPU and memory usage
- iOS and Android photo backup (using the circled.me app)
  - Supports either locally mounted disks or
  - S3-compatible Services - this allows different users to use their own S3 bucket on the same server
- iOS Push notifications for new Albums, Photos (in progress)
- Albums
  - Adding local server contributors and viewers
  - Sharing albums with anyone with a "secret" link
- Filtering photos by year, month, location, etc
- Moments - automatically grouping photos by time and location
- Reverse geocoding for all assets
- Automatic video conversion to web-compatible H.264 format


## Feautres that are in-progress and/or prioritised:
- Map browsing of photos
- Group chats
- Face detection and tagging
- Bulk-adding assets by:
  - Scanning directories on local disks
  - Scanning objects on already existing S3 bucket prefix

## Compiling and Running the server
To compile, please use Go 1.20.5 or above.
The easiest way to try and run the server is within a docker container. There's no image provided on Docker Hub (yet) so needs to be built locally, see example docker-compose file below.

```bash
git clone https://github.com/circled-me/server.git
cd server
docker-compose -f docker-compose-example.yaml up
```

Current configuration environment variables:
- `MYSQL_DSN` - see example or refer to https://github.com/go-sql-driver/mysql#dsn-data-source-name
- `BIND_ADDRESS` - IP and port to bind to (incompatible with `TLS_DOMAINS`). This is useful if your server is, say, behind reverse proxy
- `TLS_DOMAINS` - a list of comma-separated domain names. This uses the Let's Encrypt Gin implementation (https://github.com/gin-gonic/autotls)
- `DEBUG_MODE` - currently defaults to `yes`

## docker-compose example
This `docker-compose` file is **just an example** and does provide only basic configuration. 
Modify the `mysql-data` and `asset-data` below at the very least to suitable locations with enough space, etc.
Better though, use your "proper" MySQL server instead of running it in Docker.

**NOTE: Please do not use this in production.**

```yaml:
version: '2'
services:
  mysql:
    image: mysql:5.7
    command: --default-authentication-plugin=mysql_native_password
    restart: always
    volumes:
      - <mysql-data-dir>:/var/lib/mysql
    environment:
      MYSQL_DATABASE: circled
      MYSQL_ALLOW_EMPTY_PASSWORD: yes
    healthcheck:
      test: mysqladmin ping --silent
      start_period: 5s
      interval: 3s
      timeout: 5s
      retries: 20

  circled-server:
    # image: circled-server:latest
    build:
      dockerfile: Dockerfile
    restart: always
    depends_on:
      mysql:
        condition: service_healthy
    ports:
      - "8080:8080"
    environment:
      MYSQL_DSN: "root:@tcp(mysql:3306)/circled?charset=utf8mb4&parseTime=True&loc=Local"
      BIND_ADDRESS: 0.0.0.0:8080
    volumes:
      - <asset-data-dir>:/mnt/data1
```
