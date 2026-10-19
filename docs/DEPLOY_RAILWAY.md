# Deploy to Railway

Three services in one Railway project:

| Service | Source | Public? | What it does |
|---------|--------|---------|--------------|
| **MySQL** | Railway database | No | Data |
| **api** | `backend/` (Dockerfile) | No | Go API, port 8080 |
| **web** | `frontend/` (Dockerfile) | Yes (Railway domain) | Nuxt app, port 3000. Serves the app and forwards `/api/v1/*` to **api** over Railway's private network |

The browser only talks to **web**. Keeping the API on the same origin keeps the sign-in refresh
cookie first-party, so reloading stays signed in (no CORS, no cross-site cookie problems).

## What happens on deploy

- `backend/railway.toml` / `frontend/railway.toml` tell Railway to build each folder's Dockerfile and health-check it.
- The API starts with `AUTO_MIGRATE=true`: it creates/updates the tables and seeds permissions, roles,
  categories and payment methods. No demo users unless `SEED_DEMO=true`.
- The first Super Admin is created once with `POST /api/v1/auth/setup` and the `SETUP_TOKEN`.

## Deploy with the CLI (this folder is not a Git repo, so `railway up` uploads it)

`backend/.railwayignore` and `frontend/.railwayignore` keep local `.env` files (your local DB password
and JWT secret) and `node_modules` out of the upload.

```bash
railway login
```

```bash
railway init --name income-expense
```

```bash
railway add --database mysql
```

```bash
railway add --service api
```

```bash
railway add --service web
```

API variables (`openssl rand -hex 32` makes the secrets; keep them private):

```bash
railway variable set --service api --skip-deploys APP_ENV=production AUTO_MIGRATE=true PORT=8080 TZ=Asia/Phnom_Penh 'DB_HOST=${{MySQL.MYSQLHOST}}' 'DB_PORT=${{MySQL.MYSQLPORT}}' 'DB_USER=${{MySQL.MYSQLUSER}}' 'DB_PASSWORD=${{MySQL.MYSQLPASSWORD}}' 'DB_NAME=${{MySQL.MYSQLDATABASE}}' 'DB_PARAMS=charset=utf8mb4&parseTime=True&loc=Asia%2FPhnom_Penh' COOKIE_SECURE=true COOKIE_SAME_SITE=Strict "JWT_SECRET=$(openssl rand -hex 32)" "SETUP_TOKEN=$(openssl rand -hex 16)"
```

Web variables:

```bash
railway variable set --service web --skip-deploys PORT=3000 'NUXT_API_PROXY_TARGET=http://${{api.RAILWAY_PRIVATE_DOMAIN}}:8080'
```

Deploy both (run from the project root):

```bash
railway up backend --path-as-root --service api --detach
```

```bash
railway up frontend --path-as-root --service web --detach
```

Give the web service a public address:

```bash
railway domain --service web
```

Then set `CORS_ORIGINS` on the API to that address (not needed for same-origin calls, but keeps the
API's allow-list accurate):

```bash
railway variable set --service api CORS_ORIGINS=https://<your-web-domain>
```

## Create the first admin

Read the token with `railway variable list --service api --kv | grep SETUP_TOKEN`, then:

```bash
curl -X POST https://<your-web-domain>/api/v1/auth/setup -H 'Content-Type: application/json' -d '{"setup_token":"<SETUP_TOKEN>","username":"admin","email":"you@example.com","password":"1234","full_name":"Your Name"}'
```

`password` is the 4-digit PIN. Setup works only while no Super Admin exists. Afterwards remove the
token so it can't be reused:

```bash
railway variable delete --service api SETUP_TOKEN
```

Sign in at `https://<your-web-domain>`, then create branches, users and roles in the app.

## Optional: demo data

Set `SEED_DEMO=true` on **api** to load the demo branches and users (PIN `2468`, see `docs/DEMO_DATA.md`).
Those PINs are public, so only do this for a private test project, and remove the variable afterwards.

## Updating

Run the two `railway up …` commands again. Schema changes are applied by `AUTO_MIGRATE` on start.

## Dashboard alternative (GitHub)

Push this project to GitHub, then in Railway: **New Project → Deploy from GitHub repo** twice, with
**Root Directory** `/backend` for **api** and `/frontend` for **web**; add **Database → MySQL**; set the
same variables as above; **Settings → Networking → Generate Domain** on **web** only.
