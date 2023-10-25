FROM --platform=${BUILDPLATFORM} golang:alpine AS mods
ENV CGO_ENABLED=0 GOOS=linux
WORKDIR /src
RUN --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=cache,target=/go/pkg \
    go mod download

FROM --platform=${BUILDPLATFORM} mods as builder
ARG VERSION="1.0.0"
ARG TARGETARCH
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w -X 'main.Version=${VERSION}'" -o /out/wg-api

FROM scratch
COPY --from=builder /out/wg-api /wg-api
ENTRYPOINT ["/wg-api"]
