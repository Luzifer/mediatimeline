FROM golang:alpine as builder

COPY . /src/mediatimeline
WORKDIR /src/mediatimeline

RUN set -ex \
 && apk add --update git \
 && go get \
 && go install -ldflags "-X main.version=$(git describe --tags --always || echo dev)"

FROM alpine:latest

ENV DATABASE=/data/tweets.db \
    FRONTEND=/usr/local/share/mediatimeline/frontend

LABEL maintainer "Knut Ahlers <knut@ahlers.me>"

RUN set -ex \
 && apk --no-cache add ca-certificates

COPY --from=builder /go/bin/mediatimeline /usr/local/bin/mediatimeline
COPY frontend /usr/local/share/mediatimeline/frontend

EXPOSE 3000
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/mediatimeline"]
CMD ["--"]

# vim: set ft=Dockerfile:
