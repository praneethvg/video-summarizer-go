# --- Build stage ---
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY . .
RUN apk add --no-cache build-base bash curl git cmake
RUN chmod +x setup_tools.sh && ./setup_tools.sh
RUN chmod +x build_all.sh && ./build_all.sh

# --- Final stage ---
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bin ./bin
COPY --from=builder /app/models ./models
COPY --from=builder /app/prompts ./prompts
COPY --from=builder /app/tools ./tools
COPY logging.docker.yaml ./logging.yaml

EXPOSE 8080
#CMD ["./bin/service"]