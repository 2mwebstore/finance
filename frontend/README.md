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
- `useApi()` retries once after refresh on 401

## Setup

```bash
cd frontend
npm install
export NUXT_PUBLIC_API_BASE=http://127.0.0.1:8080/api/v1
npm run dev
```

Backend must allow CORS origin `http://localhost:3000` with credentials (already configured).

## Build

```bash
npm run build
npm run preview
```

## Note

Full admin screens (users, roles, branches CRUD UI, audit log viewer) are available via the API; navigation stubs live under **More** for extension.
