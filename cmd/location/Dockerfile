FROM golang:1.23-alpine AS builder

# Add necessary build tools
RUN apk add --no-cache git make

# Set build arguments
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOCACHE=/go-build-cache

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
    go build -ldflags="-s -w" -o location-service ./cmd/location/main.go

# Use a smaller image for the final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/location-service .

# Create logs directory
RUN mkdir -p /app/logs

# Set the entrypoint
ENTRYPOINT ["/app/location-service"]