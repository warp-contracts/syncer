FROM golang:1.23.4-alpine3.20
RUN apk add --update make build-base curl git

WORKDIR /app
COPY .git .git
COPY .gopath~ .gopath~
COPY main.go .
COPY go.mod .
COPY go.sum .
COPY Makefile .
COPY src src
COPY vendor vendor
RUN make version
RUN make build

FROM golang:1.23.4-alpine3.20
RUN mkdir -p /app/bin
COPY --from=0 /app/bin/syncer /app/bin/syncer

RUN mkdir -p /go/src/warpy_sync
COPY --from=0 /app/src/warpy_sync /go/src/warpy_sync

CMD ["/app/bin/syncer", "sync"]
