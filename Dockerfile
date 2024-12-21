# syntax=docker/dockerfile:1
FROM golang AS build-stage
ADD . /src
WORKDIR /src

RUN CGO_ENABLED=1 GOOS=linux go build -o /bin/twitterbridge .

FROM gcr.io/distroless/base-debian12 AS build-release-stage
COPY --from=build-stage /bin/twitterbridge /bin/twitterbridge
WORKDIR /config

ENV TWITTER_BRIDGE_DB_PATH="/config/sqlite/sqlite.db" \
    TWITTER_BRIDGE_SERVER_URL="http://127.0.0.1:3000" \
    TWITTER_BRIDGE_SERVER_PORT="3000"

CMD ["/bin/twitterbridge"]