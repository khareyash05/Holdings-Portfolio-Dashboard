// invariant: when an upstream service is paused, last-good cache must keep responses 200
// proof: run a steady 20 RPS for 60s. midway, you pause forex-service from another shell:
//   docker pause paasa-forex-service   (run at t=20s)
//   docker unpause paasa-forex-service (run at t=40s)
// during the outage window, forex calls fail, but the lastgood-forex cache (24h ttl) supplies stale rates
// threshold: zero 5xx for the entire run, including the outage window
// if last-good fallback regresses, this immediately fails with a flood of 500s

import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = 'http://localhost:8080';
const fiveXX = new Counter('responses_5xx');

export const options = {
  scenarios: {
    steady: {
      executor: 'constant-arrival-rate',
      rate: 5,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 5,
      maxVUs: 10,
    },
  },
  thresholds: {
    responses_5xx:     ['count==0'],
    http_req_failed:   ['rate<0.01'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/portfolio?base=USD`);
  if (res.status >= 500) fiveXX.add(1);
  check(res, {
    'status is 200':    (r) => r.status === 200,
    'has summary':      (r) => r.json('summary.netWorth') !== undefined,
    'holdings present': (r) => Array.isArray(r.json('holdings')) && r.json('holdings').length > 0,
  });
}
