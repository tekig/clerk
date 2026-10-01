FROM --platform=$BUILDPLATFORM golang:1.25 AS build

ARG TARGETOS
ARG TARGETARCH

# Go caches are set explicitly and used as cache mount targets
ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/go/cache

WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=$GOMODCACHE \
    go mod download

COPY . ./

RUN --mount=type=cache,target=$GOMODCACHE \
    --mount=type=cache,target=$GOCACHE \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --system --uid 10001 --user-group --no-create-home clerk

WORKDIR /app

COPY --from=build /out/ /app/bin/
COPY config/example_*.yaml /app/config/

EXPOSE 6060
EXPOSE 8080
EXPOSE 50051

ENV PATH="/app/bin/:$PATH"

USER clerk
