# Production Deployment Guide

**Stack:** Linux VPS · Docker Compose · Nginx · MySQL 8 · Go API · Nuxt 3  
**Timezone:** Asia/Phnom_Penh

## 1. Server prerequisites

- Ubuntu 22.04/24.04 (or similar)
- Docker Engine + Docker Compose plugin
- Firewall: allow 80/443 only (SSH restricted)
- Domain DNS A/AAAA pointing at the VPS

```bash
# Example firewall
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

## 2. Deploy application files

```bash
sudo mkdir -p /opt/finance
sudo chown $USER:$USER /opt/finance
# copy repository contents to /opt/finance
cd /opt/finance
cp .env.prod.example .env.prod
# edit .env.prod — set strong unique secrets
chmod 600 .env.prod
```

Required secrets in `.env.prod`:

| Variable | Purpose |
|----------|---------|
| `MYSQL_ROOT_PASSWORD` | MySQL root |
| `DB_PASSWORD` | App DB user |
| `JWT_SECRET` | ≥32 random characters |
| `SETUP_TOKEN` | One-time Super Admin setup (clear after use) |
| `CORS_ORIGINS` | `https://your-domain` |

**Never commit `.env.prod`.**

## 3. Start the stack

```bash
cd /opt/finance
docker compose -f docker-compose.prod.yml --env-file .env.prod build
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
curl -s http://127.0.0.1/health
```

MySQL is **not** published on the host network. Only Nginx exposes 80/443.

## 4. First-run Super Admin

```bash
curl -s -X POST https://YOUR_DOMAIN/api/v1/auth/setup \
  -H 'Content-Type: application/json' \
  -d '{
    "setup_token":"<SETUP_TOKEN from .env.prod>",
    "username":"superadmin",
    "email":"admin@yourcompany.com",
    "password":"<6 digit PIN>",
    "full_name":"Super Admin"
  }'
```

Then **remove or empty `SETUP_TOKEN`** in `.env.prod` and recreate the API container:

```bash
# edit .env.prod → SETUP_TOKEN=
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d api
```

## 5. HTTPS (Let’s Encrypt)

On the host (with port 80 reachable):

```bash
docker run --rm -it \
  -v finance_certbot_www:/var/www/certbot \
  -v finance_certbot_certs:/etc/letsencrypt \
  certbot/certbot certonly --webroot -w /var/www/certbot \
  -d finance.example.com --email you@example.com --agree-tos
```

Uncomment the TLS `server { ... }` block in `deploy/nginx/conf.d/app.conf`, set `server_name`, then:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod exec nginx nginx -s reload
```

Set `COOKIE_SECURE=true` (already default in prod compose) and `CORS_ORIGINS=https://finance.example.com`.

## 6. Database migrations

- Versioned SQL files live in `backend/migrations/`.
- Development uses GORM AutoMigrate when `APP_ENV=development`.
- **Production should prefer applying versioned SQL** (golang-migrate, or manual) before/at deploy.
- Never auto-destroy production data.

Example with `mysql` client against the container:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod exec -T mysql \
  mysql -ufinance -p"$DB_PASSWORD" income_expense < backend/migrations/000001_init_auth.up.sql
```

(Apply in order; skip already-applied files.)

## 7. Backups

Script: `deploy/scripts/backup-mysql.sh`

```bash
chmod +x deploy/scripts/*.sh
# cron example — 02:00 daily
0 2 * * * ROOT_DIR=/opt/finance /opt/finance/deploy/scripts/backup-mysql.sh >> /var/log/finance-backup.log 2>&1
```

- Format: `income_expense_YYYYMMDDTHHMMSSZ.sql.gz`
- Default retention: 14 days
- Optional off-server: set `RSYNC_TARGET=user@backup-host:/path`

### Restore

```bash
# STOP writes / put app in maintenance if possible
./deploy/scripts/restore-mysql.sh /var/backups/finance-mysql/income_expense_....sql.gz
```

**RPO target:** ≤ 24h (daily backups) — tighten with more frequent dumps if required.  
**RTO target:** restore + verify health ≤ 1–2 hours for a single VPS.

Test restore on staging at least quarterly.

## 8. Health checks

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Process up |
| `GET /ready` | DB connectivity |

Compose healthchecks restart unhealthy containers per Docker policy.

## 9. CI/CD (GitHub Actions)

Workflow: `.github/workflows/ci.yml`

1. Backend: `go vet`, unit tests, build  
2. Optional integration tests with MySQL service  
3. Frontend: `npm install` + `npm run build`  
4. Docker image build for API on push  

**Production promotion** should require manual approval (add an environment protection rule in GitHub). Do not store production secrets in the repository — use GitHub Secrets / a vault.

## 10. Operational checklist

- [ ] Strong unique passwords and JWT secret  
- [ ] `SETUP_TOKEN` cleared after admin created  
- [ ] HTTPS enabled; HSTS on  
- [ ] MySQL not exposed publicly  
- [ ] Automated backups running; restore tested  
- [ ] Firewall limited to 22/80/443  
- [ ] Log rotation configured on host  
- [ ] Staging environment separate from production  

## 11. Known limitations (honest)

- Production schema migration automation (golang-migrate binary in image) is not fully automated in this compose file — apply SQL migrations deliberately.
- Frontend production image requires a successful `npm run build` (lockfile recommended via `npm install` locally then commit `package-lock.json`).
- Excel/PDF export beyond CSV is not implemented yet.
- Full admin UI pages (users/roles UI) are API-complete; Nuxt screens are MVP (dashboard/transactions/reports).
EOF

cat > /home/workdir/artifacts/docs/PROGRESS.md << 'EOF'
# Progress Tracker

**Last updated**: 2026-10-17 (Phase 11 deployment artifacts)

## Phase Status

| Phase | Status |
|-------|--------|
| 1–7 Backend core | **PASS** (tested) |
| 9 Frontend MVP source | Present (npm build env-limited) |
| 11 Production deployment | **Artifacts complete** |

## Phase 11 — Delivered

| Artifact | Path |
|----------|------|
| Prod compose | `docker-compose.prod.yml` |
| API Dockerfile | `backend/Dockerfile` (multi-stage, non-root) |
| Web Dockerfile | `frontend/Dockerfile` |
| Nginx | `deploy/nginx/` |
| Env template | `.env.prod.example` |
| MySQL backup | `deploy/scripts/backup-mysql.sh` |
| MySQL restore | `deploy/scripts/restore-mysql.sh` |
| CI workflow | `.github/workflows/ci.yml` |
| Deployment guide | `docs/DEPLOYMENT.md` |

### Design choices
- MySQL internal network only (not published)
- Nginx terminates HTTP(S); proxies `/api/` → API, `/` → Nuxt
- Secrets via env file, not baked into images
- Healthchecks on mysql, api, web
- Backup gzip + retention; optional rsync off-server

### Not verified in this sandbox
- Full `docker compose up` (Docker daemon not available)
- Live TLS issuance
- End-to-end restore drill on a real VPS

Operators should follow `docs/DEPLOYMENT.md` on a real Linux VPS to complete verification.
EOF

# ensure go.sum exists for docker build
ls /home/workdir/artifacts/backend/go.sum 2>&1 | head -1
ls /home/workdir/artifacts/deploy/scripts/
ls /home/workdir/artifacts/.github/workflows/
