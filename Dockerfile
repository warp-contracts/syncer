FROM golang:1.19.4-alpine3.17
RUN apk add --update make 

WORKDIR /app
COPY .gopath~ .gopath~
COPY main.go .
COPY go.mod .
COPY go.sum .
COPY Makefile .
RUN go mod tidy
COPY src src
RUN make build

CMD ["/app/syncer", "sync"]
