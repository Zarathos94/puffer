# Puffer: pufETH/ETH Rate Service

Puffer is a full-stack service for tracking and visualizing the pufETH/ETH conversion rate over time. It fetches on-chain data, caches it, and exposes a simple API and frontend dashboard.

---

## Features

- **Backend API (Go):**  
  - Fetches and caches pufETH/ETH rates and supply from Ethereum.
  - Provides REST and SSE endpoints for current and historical rates.
  - Uses Redis for fast caching.

- **Frontend (React, with optional Vue):**  
  - Displays live and historical pufETH/ETH rates in a modern dashboard.
  - Connects to the backend API for data.

- **Dockerized:**  
  - One-command startup for API, Redis, and frontend using Docker Compose.

---

## Project Structure

```
.
├── main.go                # Go backend entrypoint
├── Dockerfile             # Backend Docker build
├── docker-compose.yml     # Multi-service orchestration
├── cache/                 # Redis cache logic
├── models/                # Data models (e.g., RateUpdate)
├── routes/                # API route handlers
├── utils/                 # On-chain logic, formatting, Etherscan helpers
├── etherscanclient/       # Etherscan API client
├── fe_react/              # React frontend (default)
└── fe_vue/                # Vue frontend (optional)
```

---

## API Endpoints

- `GET /rate` — Latest pufETH/ETH rate and supply.
- `GET /rate/history` — 24h historical rates (hourly).
- `GET /sse/rate` — Live updates via Server-Sent Events (SSE).

**Sample Response:**
```json
{
  "timestamp": 1712345678,
  "rate": 1.002345,
  "assets": "53.25K",
  "total_supply": "53.25K"
}
```

---

## Quickstart (Docker Compose)

1. **Clone the repository:**
   ```sh
   git clone <repo-url>
   cd puffer
   ```

2. **Configure environment variables:**
   - Edit `docker-compose.yml` to set your own `ETH_RPC_URL` and `ETHERSCAN_API_KEY` if needed.

3. **Start all services:**
   ```sh
   docker-compose up --build
   ```

4. **Access the dashboard:**
   - Open [http://localhost:3000](http://localhost:3000) for the React frontend.
   -  Open [http://localhost:3010](http://localhost:3010) for the Vue frontend.
   - The API is available at [http://localhost:8080](http://localhost:8080).

---

## Development

### Backend (Go API)
```sh
go run main.go
```
- Requires Redis running locally (`docker run -p 6379:6379 redis:7-alpine`).
- Set `ETH_RPC_URL`, `REDIS_ADDR`, and `ETHERSCAN_API_KEY` as environment variables.

### Frontend (React)
```sh
cd fe_react
npm install
npm run dev
```
- Set `VITE_API_URL` in `fe_react/.env` to your API endpoint (default: `http://localhost:8080`).

### Frontend (Vue, optional)
```sh
cd fe_vue
npm install
npm run dev
```
- Set `VITE_API_URL` in `fe_vue/.env` if using Vue.

---

## Customization

- **Formatting:**  
  Large ETH values are displayed with K/M/B suffixes (e.g., `53.25K` for 53,250 ETH).
- **Switching Frontends:**  
  By default, Docker Compose runs the React frontend. To use Vue, adjust the compose file accordingly.

---

## License

MIT
