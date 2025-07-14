# Build stage
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy only the necessary directories and files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the application
RUN go build -o /app/buddy-mcp ./cmd/buddy-mcp

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl jq

# Create non-root user
RUN addgroup -g 1000 buddy && \
    adduser -D -u 1000 -G buddy buddy

# Set working directory
WORKDIR /home/buddy

# Copy binary from builder
COPY --from=builder /app/buddy-mcp /usr/local/bin/buddy-mcp

# Create default buddy directory structure
RUN mkdir -p .buddy/{rules,knowledge,todos,database,history,backups} && \
    chown -R buddy:buddy .buddy

# Switch to non-root user
USER buddy

# Set environment variables
ENV BUDDY_PATH=/home/buddy/.buddy

# MCP servers communicate via stdin/stdout, no port needed
# CMD runs the MCP server
CMD ["buddy-mcp"] 