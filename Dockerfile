FROM golang:1.19 AS builder

WORKDIR /go/src/github.com/jamescun/wg-api
COPY . /go/src/github.com/jamescun/wg-api

RUN CGO_ENABLED=0 GOOS=linux go build -o wg-api main.go


FROM scratch
COPY --from=builder /go/src/github.com/jamescun/wg-api/wg-api /wg-api
ENTRYPOINT ["/wg-api"]
