While load-testing /api/portfolio/stream with k6 — 100 concurrent VUs holding for 20s at a 1s push interval — I was
  watching two things: TTFB (p95) and egress bytes per stream.

  ▎ The TTFB was fine (~120ms p95), but sse_bytes_per_stream was the surprise. Each portfolio JSON event is ~8–10 KB (holdings
   array with ticker, qty, bought, current, value, P/L per row, plus totals). At 1 event/sec for 20s, that's ~180 KB per
  connection. Across 100 VUs, one 20s run pushed ~18 MB out the wire. Linearly extrapolating to "10k concurrent users on a 5s
  interval for an hour" — which is the kind of number that comes up in the scaling discussion — that's hundreds of GB/hour of
  egress. On a managed gateway, egress is the bill, not CPU.



  1. Entry-level test → discovered cache stampede (throughput.js)
  
    ▎ The first thing I ran was throughput.js: constant 200 RPS against /api/portfolio?base=... for 30s. Pure smoke test — does
    the box hold?
  
    ▎ What broke: p99 was ugly — spikes to 800ms+ even though the median was ~30ms. Tailing logs showed why: every time the
    price cache TTL expired (3s), I'd see a burst of identical GET /exchange/.../snapshots calls fire at the exchange-service
    simultaneously. Classic thundering herd / dogpile.
  
    ▎ Diagnosis: with 5 currency bases and ~10 exchanges, I had ~50 cache keys. Every 3s, the keys expire roughly together, and
    200 in-flight requests all miss the cache and stampede the upstream.
  
    ▎ Fix: golang.org/x/sync/singleflight per snapshot key — concurrent callers for the same (exchange) collapse into one
    upstream call, the rest wait on the result. After: p99 dropped to ~120ms, upstream call volume dropped roughly 10x at the
    same RPS. This is the snapSF group in service.go.
  
    2. Same test with upstream flakiness → graceful degradation (last-good cache)
  
    ▎ While running throughput.js, I'd docker pause forex-service to see what happened. Without protection, every ?base=USD
    request started 500-ing — forex unavailable kills the whole portfolio response, even though prices are fine.
  
    ▎ Fix: dual-cache pattern — short TTL for "fresh" (60s for forex, 3s for price) and a 24h lastgood-* cache that's only read
    on upstream failure. So if forex flatlines for 5 minutes, the portfolio keeps rendering with slightly-stale rates instead of
     a red dashboard.
  
    ▎ The load-test signal that confirmed it: http_req_failed stayed at 0 even with the upstream paused. The number you don't
    see is the number that matters here — I would have shipped without this if I hadn't deliberately fault-injected during the
    run.
  
    3. Sustained-VU test → DB hot path (portfolio.js)
  
    ▎ portfolio.js is the gentler one — 20 VUs ramping over 15s, holding 30s, draining. I wasn't expecting issues but I tailed
    Postgres pg_stat_statements while it ran.
  
    ▎ What I found: SELECT * FROM exchanges was the #1 query by call count. Every portfolio request was re-fetching the
    exchanges table even though it changes ~never. Pure waste — the exchanges table is seed data.
  
    ▎ Fix: in-memory TTL cache (5min) inside Service.Exchanges(), with double-checked locking (RLock for the fast path, upgrade
    to Lock only when the entry's expired). After: that query basically disappeared from pg_stat_statements.
  
    ▎ Worth noting: I used double-checked locking specifically because the read path is hot — RWMutex lets unlimited concurrent
    readers through when the cache is fresh, which is the 99% case.
  
    4. SSE test → wire-bytes problem (sse.js, 100 VUs)
  
    ▎ (the gzip story from before) — bytes-per-stream measured at ~180KB → ~58KB after gzip BestSpeed. ~3.1x compression,
    negligible CPU.
  
    ▎ The reason this needed its own test file is that throughput.js can't catch it — short-lived requests don't expose the
    egress problem. You only see it when connections are held open and you can measure cumulative bytes per session.
  
    5. Adversarial test → rate limit tuning (ratelimit.js)
  
    ▎ ratelimit.js fires 30 RPS from a single VU at /api/portfolio for 30s. The limiter is 10 RPS, burst 20.
  
    ▎ What it confirmed: the first ~20 requests sail through (burst), then it settles into ~10 200s/sec and ~20 429s/sec for the
     rest of the run. The handleSummary at the bottom prints the exact 200/429 split — that ratio is the proof the token bucket
    is working.
  
    ▎ Why this test exists at all: I wanted to verify a single misbehaving client can't drown out everyone else, and to actually
     tune the RPS/burst. My first guess was 5 RPS / burst 10 — too aggressive, real users with a few open tabs would get 429s.
    10/20 was the sweet spot from running this script with different limiter values.



     ▎ "For each non-trivial design decision, I wrote the load test first to define the invariant, ran it red, then built until
      it went green."
    
      ▎ - Before adding singleflight, I wrote singleflight.js — fired 100 concurrent identical requests and watched p99 walk past
      1s. That's what motivated the dedup work. After singleflight, p99 dropped under 250ms — that's now the threshold guarding
      the regression.
      ▎ - Before the last-good cache, I wrote fallback.js and ran it while docker pause-ing forex. Got a flood of 500s. The
      threshold responses_5xx == 0 was set first; the last-good cache is what made it pass.
      ▎ - cache.js is the regression guard for caching itself — if someone later "simplifies" by removing Redis, the p50 threshold
       trips immediately.
      ▎ - mixed.js is the isolation test — written because I wanted to be sure the SSE work didn't quietly degrade the REST path.
      It's the kind of bug you only catch when the workloads run together.
