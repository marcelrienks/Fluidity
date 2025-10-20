# Agent Dockerfile
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

# Build the agent
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fluidity-agent ./cmd/agent

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/fluidity-agent .

# Copy configuration files
COPY configs/agent.yaml ./config/agent.yaml

# Create certs directory (certificates will be mounted or copied)
RUN mkdir -p ./certs

# Expose proxy port
EXPOSE 8080

# Command to run
CMD ["./fluidity-agent", "--config", "./config/agent.yaml"]