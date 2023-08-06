# Build container
FROM golang:buster AS builder

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates git zlib1g-dev

COPY . /go/src/github.com/TicketsBot/subscriptions-app
WORKDIR /go/src/github.com/TicketsBot/subscriptions-app

RUN set -Eeux && \
    go mod download && \
    go mod verify

RUN GOOS=linux GOARCH=amd64 \
    go build \
    -tags=jsoniter \
    -trimpath \
    -o main cmd/app/main.go

# Prod container
FROM ubuntu:latest

RUN apt-get update && apt-get upgrade -y && apt-get install -y ca-certificates curl

COPY --from=builder /go/src/github.com/TicketsBot/subscriptions-app/main /srv/subscriptions-app/main

RUN chmod +x /srv/subscriptions-app/main

RUN useradd -m container
USER container
WORKDIR /srv/subscriptions-app

CMD ["/srv/subscriptions-app/main"]