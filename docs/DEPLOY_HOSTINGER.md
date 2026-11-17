# Moving to a Hostinger VPS (in /var/www/finance)

The whole app runs on one VPS from [`docker-compose.yml`](../docker-compose.yml), in one folder:

```
/var/www/finance/            the repository (git clone), with docker-compose.yml
├── .env                     your settings and secrets (chmod 600, never in git)
└── data/                    everything the app stores (never in git or in a Docker build)
    ├── mysql/               the database
    ├── app/backups/         the app's backups (Backups page)
    ├── app/uploads/         uploaded files: receipts, product pictures…
    └── caddy/, caddy-config/  HTTPS certificates
```

| Container | What it does |
|-----------|--------------|
| `mysql`   | The database (MySQL 8.4). Not reachable from outside the server. |
| `app`     | The website, API and nginx in one image, built from the repository's `Dockerfile` (the same image Railway runs). |
| `caddy`   | Answers on ports 80/443 for your domains and gets their HTTPS certificates by itself. |

Commands below run on the VPS as root (hPanel → VPS → Terminal, or `ssh root@YOUR_VPS_IP`).
Docker Manager in hPanel still shows the containers and their logs.

## 1. The VPS

- A Hostinger VPS with Docker: the "Ubuntu 24.04 with Docker" template, or open **Docker Manager**
  once so it installs Docker. KVM 1 (4 GB RAM) or bigger: building the image needs about 2 GB.
- If the VPS firewall is on (hPanel → VPS → Security → Firewall), allow TCP 80 and 443 (and UDP 443).
- Nothing else may use ports 80/443 or serve `/var/www` as a website (no Apache or nginx on the host):
  `.env` and `data/` would be exposed. Check with `ss -ltnp | grep -E ':(80|443) '` (should print nothing).

## 2. Get the code into /var/www/finance

Public repository:

```bash
mkdir -p /var/www && git clone https://github.com/2mwebstore/finance.git /var/www/finance
```

Private `finance-pro` instead: give the VPS a read-only deploy key, then clone with it.

```bash
ssh-keygen -t ed25519 -C "finance-pro deploy" -N "" -f ~/.ssh/finance-pro && cat ~/.ssh/finance-pro.pub
```

On GitHub: finance-pro → Settings → Deploy keys → **Add deploy key**, paste that line (leave "Allow write
access" off). Then:

```bash
mkdir -p /var/www && GIT_SSH_COMMAND="ssh -i ~/.ssh/finance-pro -o IdentitiesOnly=yes" git clone git@github.com:2mwebstore/finance-pro.git /var/www/finance
```

```bash
git -C /var/www/finance config core.sshCommand "ssh -i ~/.ssh/finance-pro -o IdentitiesOnly=yes"
```

## 3. The settings

```bash
cd /var/www/finance && cp deploy/hostinger.env.example .env && chmod 600 .env
```

Make three random secrets (run it three times, here or on your Mac):

```bash
openssl rand -hex 32
```

Then fill in `.env` (`nano .env`):

| Variable | Value |
|----------|-------|
| `DOMAINS` | `rp001.pro, rph001.pro` |
| `CORS_ORIGINS` | `https://rp001.pro,https://rph001.pro` |
| `MYSQL_ROOT_PASSWORD` | a secret |
| `DB_PASSWORD` | a secret |
| `JWT_SECRET` | a secret |
| `SETUP_TOKEN` | another secret, only until step 6 |
| `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID` | optional: backups to Telegram |

Keep a copy in your password manager. The two database passwords are set when the database is first
created; changing them in `.env` later doesn't change the database's.

## 4. Start it

```bash
cd /var/www/finance && docker compose up -d --build
```

The first build takes a few minutes. Then:

```bash
docker compose ps
```

All three should be `Up` (mysql and app `healthy` after a minute). The app's log ends with
`starting income-expense-api on :8081`:

```bash
docker compose logs --tail=30 app
```

## 5. Point the domains at the VPS

At your domains' DNS, set the **A record** of `rp001.pro` and of `rph001.pro` (and `www` if you use it,
adding it to `DOMAINS` and `CORS_ORIGINS` too, then `docker compose up -d`) to the VPS's IP address.
If the domains use Cloudflare's proxy (orange cloud), set them to **DNS only** at least until the
certificates are issued. Caddy keeps retrying and gets them a minute or two after DNS points here
(`docker compose logs caddy` shows it).

From the moment DNS changes, visitors reach the new server, whose database is empty until step 7.
Do steps 5–7 at a quiet time, and right before step 5 make a fresh backup on the old site
(Backups → **Backup now** → Download) so the move carries the latest data.

## 6. The first admin (temporary)

When `https://rp001.pro/health` answers from the new server, create an admin with the setup token
(fill in your token, and a 4–6 digit PIN):

```bash
curl -s -X POST https://rp001.pro/api/v1/auth/setup -H 'Content-Type: application/json' -d '{"setup_token":"YOUR_SETUP_TOKEN","username":"setupadmin","email":"setup@example.com","password":"YOUR_PIN","full_name":"Setup Admin"}'
```

It only works once, on a database without an admin.

## 7. Restore your data

1. Sign in on `https://rp001.pro` as `setupadmin`.
2. Open **Backups**, upload the backup file (e.g. `finance-pro-backup-….tar.gz`), click its restore
   icon and type `RESTORE`.
3. The database is now the backup's: `setupadmin` is gone; sign in with your usual account. (The app
   keeps a `pre-restore` backup of what it replaced.)
4. Empty `SETUP_TOKEN` in `.env`, then:

   ```bash
   cd /var/www/finance && docker compose up -d
   ```

Check `rph001.pro` too. Once everything works, stop the old Railway service (Railway dashboard); until
then, pushes to `main` keep deploying there as well.

## 8. Updates

After pushing to GitHub:

```bash
cd /var/www/finance && git pull --ff-only && docker compose up -d --build
```

`.env` and `data/` are never touched by `git pull`. To update automatically on every push, a GitHub
Actions workflow can log in over SSH (with a key whose public half is in the VPS's
`~/.ssh/authorized_keys`) and run that same line.

## Backups

The app's daily backups (Backups page) land in `/var/www/finance/data/app/backups/` on the same server,
so also turn on **Send backups to Telegram**, and Hostinger's VPS backups/snapshots. To copy them to
your Mac:

```bash
scp 'root@YOUR_VPS_IP:/var/www/finance/data/app/backups/backup-*.tar.gz' ~/Downloads/
```

## If something is wrong

- **No HTTPS / certificate errors:** DNS doesn't point at the VPS yet, Cloudflare's proxy is on, or
  ports 80/443 are closed or taken. `docker compose logs caddy` says which.
- **`docker compose` stops at "required variable … is missing":** fill that line in `.env`.
- **Build fails on `npm ci` with a network error:** run the command again; it's a download hiccup.
- **`git pull` refuses:** a file was edited on the server; `git status` shows it.
- **Signed out after restoring:** expected; sign in with an account from the backup.
