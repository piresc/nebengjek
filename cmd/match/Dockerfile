FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o match-service ./cmd/match/main.go

# Use a smaller image for the final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/match-service .

# Create logs directory
RUN mkdir -p /app/logs

# Set the entrypoint
ENTRYPOINT ["/app/match-service"]