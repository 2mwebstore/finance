
# Stage 1: Build Nuxt 3
FROM node:22-alpine AS frontend-builder

WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

# Assumes backend/main.go is the Go entry point.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /app/api .

# Stage 3: Run both applications
FROM node:22-alpine

RUN apk add --no-cache nginx supervisor ca-certificates tzdata

WORKDIR /app

COPY --from=backend-builder /app/api /cmd/api
COPY --from=frontend-builder /frontend/.output /frontend/.output

COPY nginx.conf /etc/nginx/http.d/default.conf
COPY deploy/supervisord.conf /etc/supervisord.conf

ENV NODE_ENV=production
ENV NITRO_HOST=127.0.0.1
ENV NITRO_PORT=3000
ENV PORT=8080

EXPOSE 8080

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf", "-n"]
