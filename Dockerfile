# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for dependencies (if needed)
RUN apk add --no-cache git

# Copy go.mod and go.sum for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary statically
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o swap .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the static binary from builder
COPY --from=builder /app/swap .

# Expose port (optional but good for documentation)
EXPOSE 8080

# Run the binary
CMD ["/app/swap"]
