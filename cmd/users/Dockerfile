FROM golang:1.23-alpine AS builder

# Add necessary build tools
RUN apk add --no-cache git make

# Set build arguments
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOCACHE=/go-build-cache

# Set version argument with default
ARG VERSION=development
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the source code
COPY . .

# Build the application with optimizations
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go-build-cache \
    go build -ldflags="-s -w \
    -X main.Version=${VERSION} \
    -X main.GitCommit=${GIT_COMMIT} \
    -X main.BuildTime=${BUILD_TIME}" \
    -o users-service ./cmd/users/main.go

# Use a smaller image for the final stage
FROM alpine:latest

# Add ca-certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/users-service .

# Create logs directory
RUN mkdir -p /app/logs

# Set build info environment variables
ENV VERSION=${VERSION} \
    GIT_COMMIT=${GIT_COMMIT} \
    BUILD_TIME=${BUILD_TIME}

# Set the entrypoint
ENTRYPOINT ["/app/users-service"]