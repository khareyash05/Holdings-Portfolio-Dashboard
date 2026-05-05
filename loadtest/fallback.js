// a test to check if the last good cache works if the service is down

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
