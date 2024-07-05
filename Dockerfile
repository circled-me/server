FROM linuxserver/ffmpeg:6.1.1 AS compile

# Install Dependencies
RUN apt-get -y update && apt-get install -y --fix-missing \
    build-essential \
    cmake \
    gfortran \
    git \
    wget \
    curl \
    graphicsmagick \
    libgraphicsmagick1-dev \
    libatlas-base-dev \
    libavcodec-dev \
    libavformat-dev \
    libgtk2.0-dev \
    libjpeg-dev \
    liblapack-dev \
    libswscale-dev \
    pkg-config \
    python3-dev \
    python3-numpy \
    software-properties-common \
    zip \
    && apt-get clean && rm -rf /tmp/* /var/tmp/*

RUN apt-get install -y python3 python3-pip python3-venv

# # Virtual Environment
ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m venv $VIRTUAL_ENV
ENV PATH="$VIRTUAL_ENV/bin:$PATH"
RUN python3 -m pip install --upgrade setuptools

# # Install Dlib
ENV CFLAGS=-static
RUN pip3 install --upgrade pip && \
    git clone -b 'v19.21' --single-branch https://github.com/davisking/dlib.git && \
    cd dlib/ && \
    python3 setup.py install --set BUILD_SHARED_LIBS=OFF

RUN pip3 install face_recognition

# RUN apt-get install -y rsync
# RUN pip3 install git+https://github.com/larsks/dockerize
# RUN dockerize -t delme --no-build -o  /usr/local/bin/ffmpeg

# Circled Server Build
FROM linuxserver/ffmpeg:6.1.1 as go-server-build
# Install go
RUN apt-get update && apt-get install -y mailcap gcc
RUN apt-get install -y wget

RUN wget -c https://go.dev/dl/go1.22.0.linux-amd64.tar.gz -O - | tar -xz -C /usr/local
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=$HOME/.local/go
ENV PATH=$PATH:$HOME/go/bin

COPY go.mod /go/src/circled-server/
COPY go.sum /go/src/circled-server/
WORKDIR /go/src/circled-server/
RUN go mod download
COPY . /go/src/circled-server

RUN CGO_ENABLED=1 go build -o circled-server .



# Final clean image
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y python3 python3-pip
RUN python3 -m pip install --upgrade pip

# Copy dlib dependencies
COPY --from=compile /opt/venv /opt/venv
COPY --from=compile \
    # Sources
    /lib/x86_64-linux-gnu/libpthread.so.0 \
    /lib/x86_64-linux-gnu/libz.so.1 \
    /lib/x86_64-linux-gnu/libm.so.6 \
    /lib/x86_64-linux-gnu/libgcc_s.so.1 \
    /lib/x86_64-linux-gnu/libc.so.6 \
    /lib/x86_64-linux-gnu/libdl.so.2 \
    /lib/x86_64-linux-gnu/librt.so.1 \
    # Destination
    /lib/x86_64-linux-gnu/

COPY --from=compile \
    # Sources
    /usr/lib/x86_64-linux-gnu/libX11.so.6 \
    /usr/lib/x86_64-linux-gnu/libXext.so.6 \
    /usr/lib/x86_64-linux-gnu/libpng16.so.16 \
    /usr/lib/x86_64-linux-gnu/libjpeg.so.8 \
    /usr/lib/x86_64-linux-gnu/libstdc++.so.6 \
    /usr/lib/x86_64-linux-gnu/libxcb.so.1 \
    /usr/lib/x86_64-linux-gnu/libXau.so.6 \
    /usr/lib/x86_64-linux-gnu/libXdmcp.so.6 \
    /usr/lib/x86_64-linux-gnu/libbsd.so.0 \
    # Destination
    /usr/lib/x86_64-linux-gnu/

# "Add" python packages
ENV PATH="/opt/venv/bin:$PATH"

# Copy ffmpeg and its dependencies
COPY --from=compile /usr/local/bin/ffmpeg /usr/local/bin/ffmpeg
COPY --from=compile /lib64/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
COPY --from=compile \
    /usr/local/lib/x86_64-linux-gnu/libplacebo.so.338 \
    /usr/local/lib/x86_64-linux-gnu/libdrm.so.2 \
    /usr/local/lib/libfdk-aac.so.2 \
    /usr/local/lib/libpng16.so.16 \
    /usr/local/lib/libopencore-amrnb.so.0 \
    /usr/local/lib/libfontconfig.so.1 \
    /usr/local/lib/libshaderc_shared.so.1 \
    /usr/local/lib/libsharpyuv.so.0 \
    /usr/local/lib/libtheoraenc.so.1 \
    /usr/local/lib/libvpx.so.9 \
    /usr/local/lib/libmp3lame.so.0 \
    /usr/local/lib/libopus.so.0 \
    /usr/local/lib/libxvidcore.so.4 \
    /usr/local/lib/libfreetype.so.6 \
    /usr/local/lib/libzimg.so.2 \
    /usr/local/lib/libtheoradec.so.1 \
    /usr/local/lib/libkvazaar.so.7 \
    /usr/local/lib/libva.so.2 \
    /usr/local/lib/libvulkan.so.1 \
    /usr/local/lib/libvpl.so.2 \
    /usr/local/lib/libass.so.9 \
    /usr/local/lib/libdovi.so.3 \
    /usr/local/lib/libvdpau.so.1 \
    /usr/local/lib/libvorbis.so.0 \
    /usr/local/lib/libva-drm.so.2 \
    /usr/local/lib/libva-x11.so.2 \
    /usr/local/lib/libfribidi.so.0 \
    /usr/local/lib/libvidstab.so.1.2 \
    /usr/local/lib/libopenjp2.so.7 \
    /usr/local/lib/librav1e.so.0.7 \
    /usr/local/lib/libwebpmux.so.3 \
    /usr/local/lib/libvmaf.so.3 \
    /usr/local/lib/libopencore-amrwb.so.0 \
    /usr/local/lib/libx264.so.164 \
    /usr/local/lib/libSvtAv1Enc.so.2 \
    /usr/local/lib/libx265.so.199 \
    /usr/local/lib/libogg.so.0 \
    /usr/local/lib/libvorbisenc.so.2 \
    /usr/local/lib/libwebp.so.7 \
    /lib/x86_64-linux-gnu/libnss_compat.so \
    /lib/x86_64-linux-gnu/libnss_systemd.so.2 \
    /lib/x86_64-linux-gnu/libxml2.so.2 \
    /lib/x86_64-linux-gnu/libharfbuzz.so.0 \
    /lib/x86_64-linux-gnu/libasound.so.2 \
    /lib/x86_64-linux-gnu/libnss_compat.so.2 \
    /lib/x86_64-linux-gnu/libX11.so.6 \
    /lib/x86_64-linux-gnu/libgomp.so.1 \
    /lib/x86_64-linux-gnu/libcrypto.so.3 \
    /lib/x86_64-linux-gnu/libbrotlicommon.so.1 \
    /lib/x86_64-linux-gnu/libOpenCL.so.1 \
    /lib/x86_64-linux-gnu/libxcb-shm.so.0 \
    /lib/x86_64-linux-gnu/libX11-xcb.so.1 \
    /lib/x86_64-linux-gnu/libnss_hesiod.so \
    /lib/x86_64-linux-gnu/libv4l2.so.0 \
    /lib/x86_64-linux-gnu/libxcb-xfixes.so.0 \
    /lib/x86_64-linux-gnu/libglib-2.0.so.0 \
    /lib/x86_64-linux-gnu/libnss_files.so.2 \
    /lib/x86_64-linux-gnu/libnss_dns.so.2 \
    /lib/x86_64-linux-gnu/libm.so.6 \
    /lib/x86_64-linux-gnu/libresolv.so \
    /lib/x86_64-linux-gnu/libbsd.so.0 \
    /lib/x86_64-linux-gnu/libmvec.so.1 \
    /lib/x86_64-linux-gnu/libicuuc.so.70 \
    /lib/x86_64-linux-gnu/libXext.so.6 \
    /lib/x86_64-linux-gnu/libbrotlidec.so.1 \
    /lib/x86_64-linux-gnu/libmd.so.0 \
    /lib/x86_64-linux-gnu/libXfixes.so.3 \
    /lib/x86_64-linux-gnu/libjpeg.so.8 \
    /lib/x86_64-linux-gnu/libXau.so.6 \
    /lib/x86_64-linux-gnu/libnss_hesiod.so.2 \
    /lib/x86_64-linux-gnu/libXdmcp.so.6 \
    /lib/x86_64-linux-gnu/libresolv.so.2 \
    /lib/x86_64-linux-gnu/libexpat.so.1 \
    /lib/x86_64-linux-gnu/libxcb-dri3.so.0 \
    /lib/x86_64-linux-gnu/libz.so.1 \
    /lib/x86_64-linux-gnu/libgraphite2.so.3 \
    /lib/x86_64-linux-gnu/libxcb-shape.so.0 \
    /lib/x86_64-linux-gnu/libv4lconvert.so.0 \
    /lib/x86_64-linux-gnu/libicudata.so.70 \
    /lib/x86_64-linux-gnu/liblzma.so.5 \
    /lib/x86_64-linux-gnu/libstdc++.so.6 \
    /lib/x86_64-linux-gnu/libresolv.a \
    /lib/x86_64-linux-gnu/libgcc_s.so.1 \
    /lib/x86_64-linux-gnu/libxcb.so.1 \
    /lib/x86_64-linux-gnu/libssl.so.3 \
    /lib/x86_64-linux-gnu/libpcre.so.3 \
    /lib/x86_64-linux-gnu/libc.so.6 \
    /lib/x86_64-linux-gnu/

RUN python3 -m pip install --upgrade Pillow face-recognition
# Downgrade numpy to 1.26.4
RUN pip3 install --upgrade numpy==1.26.4

RUN apt-get clean
RUN apt-get autoclean
RUN apt-get autoremove

WORKDIR /opt/circled
COPY --from=go-server-build /go/src/circled-server/circled-server .
COPY --from=go-server-build /go/src/circled-server/templates ./templates
COPY --from=go-server-build /go/src/circled-server/faces/*.py ./faces/

# Move up
RUN apt-get install -y exiftool

ENV DEFAULT_BUCKET_DIR /mnt/down/c1
ENV SQLITE_FILE /mnt/down/c1/circled.db
ENTRYPOINT ["./circled-server"]
# ENTRYPOINT [ "/bin/bash" ]