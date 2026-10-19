# Frontend — Income & Expense (Nuxt 3)

Mobile-first Nuxt 3 + Vue 3 + TypeScript + Tailwind CSS UI for the Cambodia finance API.

## Design

- Primary blue `#2563EB`, navy `#0B1220`, canvas `#F5F8FF`
- Desktop: collapsible-style sidebar
- Mobile: bottom navigation (Home, Transactions, Add, Reports, More)

## Pages (wired to real API)

| Route | Purpose |
|-------|---------|
| `/login` | Username/email + password login |
| `/dashboard` | Six currency totals + recent txns, period filter |
| `/transactions` | List with income/expense filter |
| `/transactions/new` | Create income/expense (USD/KHR, branch, category, payment) |
| `/reports` | Summary + CSV export |
| `/more` | Profile + sign out |

## Auth model

- Access token held **in memory** (not localStorage)
- Refresh via **HttpOnly cookie** (`credentials: 'include'`)
- Global middleware redirects unauthenticated users to `/login`
- `useApi()` retries once after refresh on 401; parallel callers share one refresh request
- The API rotates the refresh cookie on each refresh and briefly accepts the previous one, so several tabs can reload at once without signing each other out. No cookie → `204` (not signed in)

## Setup

```bash
cd frontend
npm install
npm run dev
```

`frontend/.env` sets `NUXT_PUBLIC_API_BASE=/api/v1`, and `npm run dev` proxies `/api/v1` to the Go API at `API_PROXY_TARGET` (default `http://localhost:8080`), like nginx in production. Keeping the API on the app's own origin keeps the refresh cookie first-party, so reloads stay signed in in every browser. See `.env.example` for other setups.

## Build

```bash
npm run build
npm run preview
```

## Note

Full admin screens (users, roles, branches CRUD UI, audit log viewer) are available via the API; navigation stubs live under **More** for extension.
