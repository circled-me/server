FROM golang:1.20.5-alpine
COPY go.mod /go/src/circled-server/
COPY go.sum /go/src/circled-server/
WORKDIR /go/src/circled-server/
RUN go mod download
COPY . /go/src/circled-server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o circled-server .


FROM jrottenberg/ffmpeg:4.1-alpine
RUN apk --no-cache add ca-certificates exiftool
WORKDIR /root/
COPY --from=0 /go/src/circled-server/circled-server .
COPY --from=0 /go/src/circled-server/templates ./templates
ENTRYPOINT ["./circled-server"]
