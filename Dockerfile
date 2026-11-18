# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for version info and ca-certificates for HTTPS
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Build all binaries with optimizations
# -trimpath: Remove file system paths from binary
# -ldflags "-s -w": Strip debug info for smaller binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /bin/alert-menta ./cmd/main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o /bin/alert-menta-mcp ./cmd/mcp/main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o /bin/alert-menta-firstresponse ./cmd/firstresponse/main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o /bin/alert-menta-triage ./cmd/triage/main.go

# Runtime stage - using distroless for minimal attack surface
# distroless/static contains only ca-certificates and tzdata
FROM gcr.io/distroless/static-debian12:nonroot

# Labels for container metadata
LABEL org.opencontainers.image.title="alert-menta" \
      org.opencontainers.image.description="LLM-powered incident response assistant for GitHub Issues" \
      org.opencontainers.image.source="https://github.com/3-shake/alert-menta" \
      org.opencontainers.image.vendor="3-shake"

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /bin/alert-menta /usr/local/bin/
COPY --from=builder /bin/alert-menta-mcp /usr/local/bin/
COPY --from=builder /bin/alert-menta-firstresponse /usr/local/bin/
COPY --from=builder /bin/alert-menta-triage /usr/local/bin/

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Run as non-root user (65532 is the nonroot user in distroless)
USER nonroot:nonroot

# Default command
ENTRYPOINT ["alert-menta"]
CMD ["-help"]
