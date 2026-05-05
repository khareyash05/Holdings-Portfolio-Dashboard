# Holdings Portfolio Dashboard

A full-stack portfolio dashboard that displays a user's net-worth, bucket-aggregated values, and 100 holdings spread across 10 exchanges, with on-the-fly currency conversion.

- **Backend**: Go (Redis TTL Caches, SSE , Gzip Compressions, Last Cache Holding if markets close, Rate Limiting)
- **Frontend**: React + TypeScript + Vite 
- **Database**: PostgreSQL 16
- **Mock services**: `forex-service` and `exchange-service`, both in Go
- **Infra**: docker-compose

---

## Quick start (clean machine)

Prerequisites: **Docker** (with the Compose plugin) 

```bash
git clone https://github.com/khareyash05/Holdings-Portfolio-Dashboard.git
cd Holdings-Portfolio-Dashboard
docker compose up --build
```

Then open:

| URL | Service |
|---|---|
| <http://localhost:5173> | Frontend dashboard |
| <http://localhost:8080/api/portfolio?base=INR> | Backend API |
| <http://localhost:8081/rates?base=INR> | Forex service |
| <http://localhost:8082/exchange/NYSE/snapshots> | Exchange service |

To shut everything down and wipe state:

```bash
docker compose down -v
```

---

## Features

- **Live streaming via SSE** — single `EventSource` per client on `/api/portfolio/stream`; backend pushes a fresh snapshot every 3s. Browser auto-reconnects.
- **Two-tier Redis cache** — hot keys (`price:<exch>` TTL 3s, `forex:<base>` TTL 60s) for steady-state, plus a 24h "last-good" key per upstream so the dashboard keeps serving when forex/exchange are down.
- **Per-IP rate limiting** — token bucket algorimthm(`golang.org/x/time/rate`) with configurable rates per second fillup + bucket capacity, returns `429` + `Retry-After`.
- **Gzip-compressed SSE stream** — content-encoding per request;  about 3× reduction on the snapshot payload.
- **Circuit breaker on upstream calls** — if connections with other services(here- forex and exchange) has issues where more than 50% are failures(total requests should be min 3), then the connection will break, reducing the hanging connection
---

## UI Summary

* Header: brand + currency switcher (from the forex service)
* Top summary: total net-worth, total invested, total unrealized gains — all in the selected base currency.
* Buckets section: grouping selector (Exchange / Region-Country / Sector-Industry / Currency) plus ascending/descending sort. Default grouping is **Exchange**
* All Holdings table: 100 rows, sortable by Net-worth, Current Price, Quantity, Unrealized Gains, or Invested amount, ascending or descending
* The page subscribes to `/api/portfolio/stream` over Server-Sent Events; the backend pushes a fresh snapshot every 3 s so prices visibly drift without the client polling

---

## Approach

### Bought price and Current Price
Bought prices are random values written to Postgres once at first boot, then never recomputed. Ranges are currency-aware so they sit in the same band as live prices (USD/EUR `[50, 500)`, INR `[500, 5000)`, JPY `[1000, 10000)`, HKD `[50, 800)`). Live current prices come from exchange-service, which generates a deterministic per-ticker base (sha256-seeded) plus ±2% jitter on each call, using the same currency bands. Bought and current are independent random draws — gains/losses can be wide

### Currency conversion
For any requested base `B` the price is an follows

```
amount_in_B = amount_in_A × rates[B](in INR) / rates[A](in INR)
```

### Caching
- **Price cache** keyed by exchange short name, TTL = 3s. 
- **Forex cache** keyed by base currency, TTL = 60s.
- The `Exchanges()` list is loaded from Postgres once.

### Realtime
Server-Sent Events. The client opens a single `EventSource` on `/api/portfolio/stream?base=X`; the backend pushes a fresh snapshot every 3s (matching the price cache TTL).

---

## API endpoints

| Method | Path | Purpose |
|---|---|---|
| GET | `/health`, `/api/health` | Liveness probe |
| GET | `/api/portfolio?base=X` | One-shot portfolio in base currency |
| GET | `/api/portfolio/stream?base=X&interval=3s` | SSE stream; gzipped if `Accept-Encoding: gzip` |
| GET | `/api/currencies` | Supported base currencies (from forex-service) |
| GET | `/api/exchanges` | Exchange list (in-memory cached, 5min TTL) |

All `/api/*` routes are behind the per-IP token-bucket rate limiter (`RATE_LIMIT_RPS` / `RATE_LIMIT_BURST`). Limited responses return `429` with `Retry-After`.


## Load testing

Introduced [k6](https://k6.io) Load Testing in the repository for load testing using Virtual Users. The scripts are present at `loadtest/` folder.

How to Run?

```bash
k6 run loadtest/portfolio.js

# without installing k6
docker run --rm -i -e BASE_URL=http://host.docker.internal:8080 \
  grafana/k6 run - < loadtest/portfolio.js
```

Each script encodes a specific invariant; the k6 thresholds inside each file fail loudly if a regression breaks it.

| Script | Invariant proven | Notes |
|---|---|---|
| `portfolio.js` | Steady stability — 20 VUs ramp/hold/drain, p95 < 500ms | Smoke test |
| `throughput.js` | 200 RPS sustained, p95 < 100ms, p99 < 300ms | Headroom test |
| `sse.js` | 100 concurrent SSE held for 20s, TTFB p95 < 500ms | Stream stability |
| `ratelimit.js` | At 30 RPS from one IP, both 200s and 429s appear | Verifies token bucket |
| `cache.js` | Hot-path p50 < 25ms over 50 sequential same-key reads | Verifies Redis price cache |
| `fallback.js` | `responses_5xx == 0` even with forex-service `docker pause`d mid-run | Verifies last-good cache |

For `fallback.js`, drive chaos in another shell:

```bash
k6 run loadtest/fallback.js &
sleep 20 && docker pause paasa-portfolio-forex-service-1
sleep 20 && docker unpause paasa-portfolio-forex-service-1
```

An Output Example on Testing Server Side Events with 100 Virtual Users logging in at same time and the connection should be withhold. Request Timeout means that the connection was withheld for the 1m.

```
➜  portfolio-holdings-dashboard git:(main) ✗ k6 run loadtest/sse.js

         /\      Grafana   /‾‾/
    /\  /  \     |\  __   /  /
   /  \/    \    | |/ /  /   ‾‾\
  /          \   |   (  |  (‾)  |
 / __________ \  |_|\_\  \_____/


     execution: local
        script: loadtest/sse.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * sse: 1 iterations for each of 100 VUs (maxDuration: 30s, gracefulStop: 30s)

WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"
WARN[0020] Request Failed                                error="request timeout"


  █ THRESHOLDS

    sse_failed_connect
    ✓ 'count<2' count=0

    sse_ttfb_ms
    ✓ 'p(95)<500' p(95)=139.76ms


  █ TOTAL RESULTS

    CUSTOM
    sse_bytes_per_stream...: avg=675543.61 min=668138  med=669226   max=686569   p(90)=686549   p(95)=686551
    sse_failed_connect.....: 0       0/s
    sse_held_ok............: 100     4.98744/s
    sse_ttfb_ms............: avg=112.24ms  min=62.39ms med=115.89ms max=141.61ms p(90)=137.95ms p(95)=139.76ms

    HTTP
    http_req_duration......: avg=20.02s    min=20s     med=20.01s   max=20.04s   p(90)=20.04s   p(95)=20.04s
    http_req_failed........: 100.00% 100 out of 100
    http_reqs..............: 100     4.98744/s

    EXECUTION
    iteration_duration.....: avg=20.03s    min=20.01s  med=20.03s   max=20.04s   p(90)=20.04s   p(95)=20.04s
    iterations.............: 100     4.98744/s
    vus....................: 100     min=100        max=100
    vus_max................: 100     min=100        max=100

    NETWORK
    data_received..........: 68 MB   3.4 MB/s
    data_sent..............: 14 kB   688 B/s




running (0m20.1s), 000/100 VUs, 100 complete and 0 interrupted iterations
sse  ✓ [======================================] 100 VUs  20.1s/30s  100/100 iters, 1 per VU
```