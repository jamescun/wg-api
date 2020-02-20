FROM golang:1.13 AS builder

WORKDIR /go/src/github.com/jamescun/wireguard-api
COPY . /go/src/github.com/jamescun/wireguard-api

RUN CGO_ENABLED=0 GOOS=linux go build -o wireguard-api cmd/wireguard-api.go


FROM scratch
COPY --from=builder /go/src/github.com/jamescun/wireguard-api/wireguard-api /bin/wireguard-api
CMD ["wireguard-api"]
