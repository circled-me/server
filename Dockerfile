FROM golang:1.18-alpine
# RUN apk add dlib --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/
# RUN apk --no-cache add openblas openblas-dev lapack lapack-dev libjpeg-turbo-dev
# RUN apk add wget cmake make gcc libc-dev g++ unzip libx11-dev pkgconf jpeg jpeg-dev libpng libpng-dev
# # Some .so symlinks are missing, need this hack
# WORKDIR /usr/lib
# RUN ln -s libblas.so.3 libblas.so
# RUN ln -s libcblas.so.3 libcblas.so
# RUN ln -s liblapack.so.3 liblapack.so

COPY go.mod /go/src/circled-server/
COPY go.sum /go/src/circled-server/
WORKDIR /go/src/circled-server/
RUN go mod download
COPY . /go/src/circled-server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o circled-server .


FROM jrottenberg/ffmpeg:4.1-alpine
RUN apk --no-cache add ca-certificates
# RUN apk add dlib --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/
# RUN apk --no-cache add openblas lapack libjpeg-turbo libstdc++
WORKDIR /root/
COPY --from=0 /go/src/circled-server/circled-server .
COPY --from=0 /go/src/circled-server/templates ./templates
ENTRYPOINT ["/bin/sh", "-c", "GODEBUG=madvdontneed=1 ./circled-server 1>>/var/log/circled-server.log 2>>/var/log/circled-server.log"]
