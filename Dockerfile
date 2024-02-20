FROM golang:1.21.7-alpine3.19 AS builder

# RUN apk add --no-cache git ca-certificates build-base
RUN apk add --no-cache git


WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build -o frame .
RUN mv ./frame /usr/bin/frame

FROM alpine:3.19
WORKDIR /app

ENV UID=1337 \
    GID=1337

RUN apk add --no-cache ffmpeg su-exec ca-certificates olm bash jq yq curl

COPY --from=builder /usr/bin/frame /usr/bin/frame
# VOLUME /data


CMD ["/usr/bin/frame"]
