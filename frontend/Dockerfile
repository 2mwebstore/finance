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

# Debian (glibc) runtime: the server proxies /api/v1 to the API by host name
# (e.g. api.railway.internal), which Alpine's musl resolver can fail to look up.
FROM node:22-bookworm-slim
WORKDIR /app
COPY --from=builder --chown=node:node /app/.output ./.output
COPY --from=builder --chown=node:node /app/package.json ./package.json
USER node
ENV HOST=0.0.0.0
ENV PORT=3000
ENV NODE_ENV=production
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD node -e "fetch('http://127.0.0.1:' + (process.env.PORT || 3000) + '/').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))"
CMD ["node", ".output/server/index.mjs"]
