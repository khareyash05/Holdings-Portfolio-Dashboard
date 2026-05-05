# Scaling — what to enhance next

Short list of moves to scale this architecture toward 1M concurrent users. 

---

## Why SSE, not WebSockets

WebSockets would be overkill here. The dashboard is **one-way** server → browser; the client never sends mid-stream messages. SSE gives us that for free over plain HTTP, with auto-reconnect built into `EventSource`. WS would double protocol complexity for zero benefit.

### However when we scale it to a full application- we will need WebSockets

---

## What to enhance

### CDN in front of SSE + static assets
Introduce CDN for SSE and ensure that the compression happens from there. Also the Static Assets can be served there

### SSE backpressure / slow-consumer handling
Right now a slow client , maybe due to slow Wi-FI can stall its goroutine. We need a non-blocking write with drop-oldest, or close the connection if the buffer stays full for N ticks. Otherwise one slow user pins resources.

### Push instead of pull from upstream
The mock pulls every 3s. In production, we can replace it with a push mechanism ex- Web Sockets so the price-ticker writes Redis on every message.

### In-process singleflight or shared-writer ticker
Right now, if a cache miss happens, they go to search in the DB. example - 100 users asking for same stock to DB, which will increase the load unecessarily, we can introduce singleflight mechanism here, in which all concurrent similar calls get cached responses after the first one calls DB.

### Deterministic bought-price seeding
Right Now, the biggest issue with price calculation is its non determinism. It's now random, thatswhy on every application db reload, we will get random values. We can ensure that the following stock will give same price by introducing hashing using seed  -> salt + stock . Easy to implement, but for the project seemed overkill

### Holdings cache invalidation via Redis pub/sub
Today holdings are cached for 30s globally. At scale, key by user-id and invalidate on `holdings:user:{id}` will be better and easier for lookups

### DB connection pool tuning
GORM's default pool is effectively unbounded. Under burst load this can open more connections than Postgres can accept. Setting `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime` keeps the pool bounded and prevents pod restarts from blowing up the DB.

### Observability
Add Prometheus metrics (request rate, cache hit ratio, SSE conns held, upstream latency). Currently the only signal is logs.

### Postgres sharding
Defer until trades exceed ~1k/sec — at 1M users the current load is ~12 writes/sec, nowhere close. When it does matter, shard by `user_id` and route reads via PgBouncer.

### Auth and per-user holdings
v1 has a single global holdings set. Real product needs JWT (or session in Redis) + per-user `holdings.user_id`, with the per-user holdings cache above.
