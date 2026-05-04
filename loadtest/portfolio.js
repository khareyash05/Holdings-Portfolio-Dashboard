// a simple load test on portfolio, by introducing 20 virtual users , holding them for 30 s and then draining them to 0
// for analyzing graceful shutdown of sse connections under load as well as its stability

import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = 'http://localhost:8080';
const BASES = ['INR', 'USD', 'EUR', 'GBP', 'JPY'];

export const options = {
  stages: [
    { duration: '15s', target: 20 }, // ramp up
    { duration: '30s', target: 20 }, // hold
    { duration: '10s', target: 0 },  // drain
  ],
  thresholds: {
    http_req_failed:   ['rate<0.01'],   // <1% errors
    http_req_duration: ['p(95)<500'],   // p95 under 500ms
  },
};

export default function () {
  const base = BASES[Math.floor(Math.random() * BASES.length)];
  const res = http.get(`${BASE_URL}/api/portfolio?base=${base}`);
  check(res, {
    'status is 200': (r) => r.status === 200,
    'has summary':   (r) => r.json('summary.netWorth') !== undefined,
    'has holdings':  (r) => Array.isArray(r.json('holdings')),
  });
  sleep(1);
}
