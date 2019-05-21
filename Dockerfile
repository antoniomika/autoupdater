FROM golang:1.12.5-alpine as builder
LABEL maintainer="Antonio Mika <me@antoniomika.me>"

RUN apk add --no-cache git gcc musl-dev bash gnupg

ENV GOCACHE /gocache

WORKDIR /usr/local/go/src/github.com/antoniomika/autoupdater

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go install
RUN go test -i ./...

FROM alpine
LABEL maintainer="Antonio Mika <me@antoniomika.me>"

RUN apk update \
        && apk upgrade \
        && apk add --no-cache \
        ca-certificates \
        && update-ca-certificates 2>/dev/null || true

COPY --from=builder /usr/local/go/src/github.com/antoniomika/autoupdater /autoupdater
COPY --from=builder /go/bin/autoupdater /autoupdater/autoupdater.bin

WORKDIR /autoupdater

ENTRYPOINT ["/autoupdater/autoupdater.bin"]
