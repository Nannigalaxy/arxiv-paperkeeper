# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o arxiv-paperkeeper ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/arxiv-paperkeeper .

# Copy web assets
COPY --from=builder /app/web ./web

# Copy default config
COPY --from=builder /app/config.yaml .

# Create data directory
RUN mkdir -p /root/data

# Expose port
EXPOSE 8080

# Set environment variables
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080
ENV DB_PATH=/root/data/arxiv.db

# Run the application
CMD ["./arxiv-paperkeeper", "server"]
