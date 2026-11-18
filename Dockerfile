# Build stage for backend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /build
COPY server/ .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o gloski ./cmd/gloski \
    && CGO_ENABLED=0 go build -ldflags "-s -w" -o gloski-agent ./cmd/gloski-agent

# Build stage for frontend
FROM oven/bun:1-alpine AS frontend-builder

WORKDIR /build
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ .
RUN bun run build

# Final stage
FROM alpine:3.19

RUN apk add --no-cache bash git

WORKDIR /app

# Copy backend binaries
COPY --from=backend-builder /build/gloski /usr/local/bin/
COPY --from=backend-builder /build/gloski-agent /usr/local/bin/

# Copy frontend build
COPY --from=frontend-builder /build/dist /app/web

# Environment variables
ENV GLOSKI_HOST=0.0.0.0
# No tailscaled inside the container, so bind the agent port explicitly.
ENV GLOSKI_MASTER_AGENT_GRPC_HOST=0.0.0.0
ENV GLOSKI_PORT=8080
ENV GLOSKI_LOG_LEVEL=info
ENV GLOSKI_MASTER_WEB_DIR=/app/web

EXPOSE 8080

ENTRYPOINT ["gloski"]