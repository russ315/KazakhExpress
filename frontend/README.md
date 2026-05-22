# KazakhExpress Frontend

Consumer marketplace built with React, TypeScript, and Vite.

## Routes

```txt
/       storefront catalog, auth, cart, order, payment, review
/ops    separate backend health view for local demos
```

The app talks only to the API Gateway:

```txt
VITE_API_BASE_URL=http://localhost:8080
```

## Local Run

```powershell
npm install
npm run dev -- --host 127.0.0.1 --port 5173
```

## Build And Verify

```powershell
npm run lint
npm run build
```

## Docker

```powershell
docker compose up --build frontend api-gateway
```

The container serves the SPA on:

```txt
http://localhost:5173
```
