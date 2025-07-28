# Multi-stage build
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN apk add --no-cache build-base bash curl git cmake
RUN chmod +x setup_tools.sh && ./setup_tools.sh
RUN chmod +x build_all.sh && ./build_all.sh

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create app user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Create necessary directories
RUN mkdir -p /app/bin /app/models /app/prompts /app/tools /app/config /app/secrets /app/logs && \
    chown -R appuser:appgroup /app

# Copy binaries and assets from builder
COPY --from=builder --chown=appuser:appgroup /app/bin /app/bin
COPY --from=builder --chown=appuser:appgroup /app/models /app/models
COPY --from=builder --chown=appuser:appgroup /app/prompts /app/prompts
COPY --from=builder --chown=appuser:appgroup /app/tools /app/tools

# Copy configuration templates
COPY --chown=appuser:appgroup config.yaml.template /app/config/config.yaml.template
COPY --chown=appuser:appgroup service.yaml.template /app/service.yaml.template
COPY --chown=appuser:appgroup sources.yaml.template /app/sources.yaml.template
COPY --chown=appuser:appgroup logging.docker.yaml /app/logging.yaml

# Set working directory
WORKDIR /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Default command
CMD ["/app/bin/service"]