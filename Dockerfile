
# Stage 1: Build Nuxt 3 frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build


# Stage 2: Build Go API
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# Correct Go entry point: backend/cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /app/api ./cmd/api


# Stage 3: Run both applications
FROM node:22-alpine

RUN apk add --no-cache nginx supervisor ca-certificates tzdata

WORKDIR /app

# Copy compiled Go binary
COPY --from=backend-builder /app/api /app/api

# Copy Nuxt SSR output
COPY --from=frontend-builder /frontend/.output /frontend/.output

# Copy service configurations
COPY nginx.conf /etc/nginx/http.d/default.conf
COPY deploy/supervisord.conf /etc/supervisord.conf

ENV NODE_ENV=production
ENV NITRO_HOST=127.0.0.1
ENV NITRO_PORT=3000
ENV PORT=8080

EXPOSE 8080

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf", "-n"]
