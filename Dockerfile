FROM golang:1.22-alpine
RUN apk add make gcc libc-dev mailcap

COPY go.mod /go/src/circled-server/
COPY go.sum /go/src/circled-server/
WORKDIR /go/src/circled-server/
RUN go mod download
RUN CGO_CFLAGS="-D_LARGEFILE64_SOURCE" go build github.com/mattn/go-sqlite3
COPY . /go/src/circled-server
RUN CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" GOOS=linux go build -a -installsuffix cgo -o circled-server .

# Final output image
FROM alpine:3.20.1
RUN apk --no-cache add ca-certificates exiftool tzdata ffmpeg
WORKDIR /opt/circled
COPY --from=0 /etc/mime.types /etc/mime.types
COPY --from=0 /go/src/circled-server/circled-server .
COPY --from=0 /go/src/circled-server/templates ./templates
ENTRYPOINT ["./circled-server"]