# Build stage for backend
FROM golang:1.21-alpine AS backend-builder

WORKDIR /build
COPY server/ .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o gloski ./cmd/gloski

# Build stage for frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /build
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Final stage
FROM alpine:3.19

RUN apk add --no-cache bash git

WORKDIR /app

# Copy backend binary
COPY --from=backend-builder /build/gloski /usr/local/bin/

# Copy frontend build
COPY --from=frontend-builder /build/dist /app/web

# Environment variables
ENV GLOSKI_HOST=0.0.0.0
ENV GLOSKI_PORT=8080
ENV GLOSKI_LOG_LEVEL=info

EXPOSE 8080

ENTRYPOINT ["gloski"]