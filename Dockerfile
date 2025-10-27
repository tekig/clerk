# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Build binaries for the target platform
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o ./bin/hammer ./cmd/hammer && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o ./bin/recorder ./cmd/recorder && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o ./bin/searcher ./cmd/searcher

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/ ./bin/

EXPOSE 6060
EXPOSE 8080
EXPOSE 50051

ENV PATH="/app/bin/:$PATH"