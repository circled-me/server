FROM golang:1.22-alpine
RUN apk add dlib dlib-dev --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/
RUN apk add blas blas-dev cblas lapack lapack-dev libjpeg-turbo-dev cmake make gcc libc-dev g++ unzip libx11-dev pkgconf jpeg jpeg-dev libpng libpng-dev mailcap

COPY go.mod /go/src/circled-server/
COPY go.sum /go/src/circled-server/
WORKDIR /go/src/circled-server/
RUN go mod download
RUN CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" GOOS=linux go build github.com/Kagami/go-face
RUN CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" GOOS=linux go build github.com/mattn/go-sqlite3
COPY . /go/src/circled-server
RUN CGO_ENABLED=1 CGO_CFLAGS="-D_LARGEFILE64_SOURCE" GOOS=linux go build -a -installsuffix cgo -o circled-server .

# Final output image
FROM alpine:3.20.1
RUN apk add dlib --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing/
RUN apk --no-cache add ca-certificates exiftool tzdata blas cblas lapack libjpeg-turbo libstdc++ libgcc ffmpeg
WORKDIR /opt/circled
# Use 68 landmarks model instead of 5 landmarks model
ADD https://github.com/ageitgey/face_recognition_models/raw/master/face_recognition_models/models/shape_predictor_68_face_landmarks.dat ./models/shape_predictor_5_face_landmarks.dat
ADD https://github.com/ageitgey/face_recognition_models/raw/master/face_recognition_models/models/dlib_face_recognition_resnet_model_v1.dat ./models/
ADD https://github.com/ageitgey/face_recognition_models/raw/master/face_recognition_models/models/mmod_human_face_detector.dat ./models/
COPY --from=0 /etc/mime.types /etc/mime.types
COPY --from=0 /go/src/circled-server/circled-server .
COPY --from=0 /go/src/circled-server/templates ./templates
COPY --from=0 /go/src/circled-server/static ./static
ENTRYPOINT ["./circled-server"]