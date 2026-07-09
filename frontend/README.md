# Camp 2026 Game Frontend

TanStack Start frontend for the Camp 2026 game.

## Commands

```sh
pnpm install
pnpm dev
pnpm test
pnpm build
```

The demo Home page calls the backend health check at `GET /api/healthz`.
Set `APP_ORIGIN` when running SSR behind a non-local origin.

## Backend API Proxy

During local development, `pnpm dev` proxies same-origin requests under
`/api` to the backend configured by `API_PROXY_TARGET`. Frontend API code should
call backend endpoints with relative paths, for example:

```ts
apiClient.get("/api/healthz")
```

Proxy settings live in `.env`. Start from the example file:

```sh
cp .env.example .env
```

The example config points to the deployed staging backend. Change
`API_PROXY_TARGET` when the backend runs on a different origin.

This dev proxy avoids browser CORS for local frontend-to-backend calls. For
production, serve the frontend and backend behind the same origin or configure
an equivalent reverse proxy at the deployment layer.

## Icons

- Use `lucide-react` for common interface and action icons.
- Use `@iconify/react` when an icon is better sourced from Iconify icon sets.
