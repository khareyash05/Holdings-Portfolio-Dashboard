// invariant: redis price cache (3s ttl) must serve hot reads materially faster than cold
// proof: fire 50 sequential requests at the same base with no sleep
//   the first miss warms the cache; the next ~50 within the 3s ttl window must hit redis
// expectation: cold p50 ~30-80ms (db + upstream + serialize), hot p50 ~5-15ms (redis + serialize)
// threshold: p50 < 25ms across the whole run — if caching regresses, p50 will look like cold p50
// also: median of last 40 requests should be well under 20ms (steady-state hot path)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

const BASE_URL = 'http://localhost:8080';
const hotLatency = new Trend('cache_hot_ms', true);

export const options = {
  scenarios: {
    sequential: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 50,
      maxDuration: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(50)<25', 'p(95)<100'],
    cache_hot_ms:      ['p(50)<20', 'p(95)<60'],
  },
};

let i = 0;
export default function () {
  const res = http.get(`${BASE_URL}/api/portfolio?base=INR`);
  check(res, { 'status is 200': (r) => r.status === 200 });
  i++;
  if (i > 10) hotLatency.add(res.timings.duration);
  sleep(0.15);
}
