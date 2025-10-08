# syntax=docker/dockerfile:1
FROM golang:1.21-trixie AS build-stage

# Install libvips and libvips-dev
RUN apt-get update && \
    apt-get install -y libvips libvips-dev && \
    rm -rf /var/lib/apt/lists/*

ADD . /src
WORKDIR /src

RUN CGO_ENABLED=1 GOOS=linux go build -o /bin/twitterbridge .


FROM gcr.io/distroless/base-debian12 AS build-release-stage
COPY --from=build-stage /bin/twitterbridge /bin/twitterbridge
WORKDIR /config

ENV TWITTER_BRIDGE_DATABASE_TYPE="sqlite" \
    TWITTER_BRIDGE_DATABASE_PATH="/config/database/sqlite.db" \
    TWITTER_BRIDGE_CDN_URL="http://127.0.0.1:3000" \
    TWITTER_BRIDGE_SERVER_PORT=3000 \
    TWITTER_BRIDGE_TRACK_ANALYTICS=true \
    TWITTER_BRIDGE_DEVELOPER_MODE=false

CMD ["/bin/twitterbridge"]