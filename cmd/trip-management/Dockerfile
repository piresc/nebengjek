FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o trip-management-service ./cmd/trip-management/main.go

# Use a smaller image for the final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/trip-management-service .

# Create logs directory
RUN mkdir -p /app/logs

# Set the entrypoint
ENTRYPOINT ["/app/trip-management-service"]