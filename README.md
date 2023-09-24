# circled.me community server
This project aims to help people easily backup and share photos, videos, albums on their own server.
Upcoming releases will further enable you to share with your circles by including group chats and more...  

After certain services became paid some time ago, I have decided I'd rather be able host my own photos.
The main reason is, of course, privacy! This project has currently only one contributor, so help will be greatly appreciated 😊

Another focus of this project is having the ability to host everything a community needs to be able to communicate and exchange photos, ideas, etc.
I strongly believe in local/focused communities and sharing with the community, but at the same time - keeping everything private, within the community.
In my personal case, I share mostly photos with my family and close friends.

## Mobile app
The **circled.me** mobile app **works with multiple accounts and servers**. For example, you can have your family server and account, and your gaming/running/basketball comunities' accounts on the same app and being able to interact with all of them at the same time.

<img src="https://app.circled.me/screenshots.jpg"/>

___

⚠️ **NOTE: Please note that this project is still in development and could introduce breaking changes.**

⚠️ **WARNING: Do not use this as your main/only backup solution.**

___


## Main features:
- Fast response times and low CPU usage
- iOS and Android photo backup (using the circled.me app)
  - Supports either locally mounted disks or
  - S3-compatible Services - this allows different users to use their own S3 bucket
- iOS Push notifications for new Albums, Photos (in progress)
- Albums
  - Adding local server contributors and viewers
  - Sharing albums with anyone with a "secret" link
- Filtering photos by year, month, location, etc
- Moments - automatically grouping photos by time and location
- Reverse geocoding for all assets
- Automatic video conversion to web-compatible H.264


## Feautres that are in-progress and/or prioritised:
- Map browsing of photos
- Group chats
- Face detection and tagging

## Running the server
The easiest way to run the server is within a docker container. 

Modify the `mysql-data` and `asset-data` at the very least to suitable locations with enough space, etc.

NOTE: Please do not use this in production. 

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
