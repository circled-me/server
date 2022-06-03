FROM golang:1.17
COPY go.mod /go/src/circled-server/
COPY go.sum /go/src/circled-server/
WORKDIR /go/src/circled-server/
RUN go mod download
COPY . /go/src/circled-server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o circled-server .


FROM alpine:3.15
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/circled-server/circled-server .
# RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf
CMD ["/bin/sh", "-c", "GODEBUG=madvdontneed=1 ./circled-server 1>>/var/log/circled-server.log 2>>/var/log/circled-server.log"]
