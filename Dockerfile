
# Stage 1: Build Vue 3 + Vite frontend
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

# Change this if your main.go is located elsewhere.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -o /app/server ./cmd/server


# Stage 3: Run both applications in one container
FROM alpine:3.22

RUN apk add --no-cache ca-certificates nginx supervisor

WORKDIR /app

COPY --from=backend-builder /app/server /app/server
COPY --from=frontend-builder /frontend/dist /usr/share/nginx/html

COPY nginx.conf /etc/nginx/http.d/default.conf
COPY deploy/supervisord.conf /etc/supervisord.conf

ENV GIN_MODE=release
ENV PORT=8080

EXPOSE 8080

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf", "-n"]
