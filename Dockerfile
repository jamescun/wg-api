FROM golang:alpine AS mods
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

FROM mods as builder
ARG VERSION="1.0.0"
COPY . .
RUN go install -ldflags "-s -w -X 'main.Version=${VERSION}'"

FROM scratch
COPY --from=builder /go/bin/wg-api /wg-api
ENTRYPOINT ["/wg-api"]
