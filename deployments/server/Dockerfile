# Server Dockerfile
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fluidity-server ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/fluidity-server .

# Copy configuration files
COPY configs/server.yaml ./config/server.yaml

# Create certs directory (certificates will be mounted or copied)
RUN mkdir -p ./certs

# Expose server port
EXPOSE 8443

# Command to run
CMD ["./fluidity-server", "--config", "./config/server.yaml"]