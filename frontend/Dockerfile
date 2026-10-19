# Multi-stage production build for Nuxt 3
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --omit=dev=false || npm install
COPY . .
ARG NUXT_PUBLIC_API_BASE=/api/v1
ENV NUXT_PUBLIC_API_BASE=$NUXT_PUBLIC_API_BASE
ENV NODE_ENV=production
RUN npm run build

FROM node:22-alpine
WORKDIR /app
RUN adduser -D -u 10001 appuser
COPY --from=builder /app/.output ./.output
COPY --from=builder /app/package.json ./package.json
USER appuser
ENV HOST=0.0.0.0
ENV PORT=3000
ENV NODE_ENV=production
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3000/ || exit 1
CMD ["node", ".output/server/index.mjs"]
