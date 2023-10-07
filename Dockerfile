FROM golang:1.20.5-alpine
COPY go.mod /go/src/circled-server/
COPY go.sum /go/src/circled-server/
WORKDIR /go/src/circled-server/
RUN go mod download
COPY . /go/src/circled-server
RUN go build -o circled-server .


FROM jrottenberg/ffmpeg:6-alpine
RUN apk --no-cache add ca-certificates exiftool tzdata
WORKDIR /opt/circled
COPY --from=0 /go/src/circled-server/circled-server .
COPY --from=0 /go/src/circled-server/templates ./templates
ENTRYPOINT ["./circled-server"]
