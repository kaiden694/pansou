# PanSou Web

Vue 3 + TypeScript + Vite frontend using the same-origin `/api` endpoints.

## Local development

```powershell
npm install
npm run dev
```

The development server proxies `/api` to `http://localhost:8888` by default. Set `VITE_PROXY_TARGET` to use another backend address.

## Verification

```powershell
npm test
npm run build
```

## Production deployment

The root Dockerfile builds the frontend and copies `dist` to `/app/web`. The Go service uses `WEB_DIST_PATH=/app/web` to serve the homepage, static assets, and client-side route fallback. Requests under `/api/*` remain backend API requests.
