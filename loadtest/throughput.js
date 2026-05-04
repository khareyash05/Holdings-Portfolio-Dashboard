// throughput test for 200 requests per second for 30 s on portfolio page 
// how much traffic can one backend actually take before it cracks

import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = 'http://localhost:8080';
const BASES = ['INR', 'USD', 'EUR', 'GBP', 'JPY'];

export const options = {
  scenarios: {
    throughput: {
      executor: 'constant-arrival-rate',
      rate: 200,
      timeUnit: '1s',
      duration: '30s',
      preAllocatedVUs: 50,
      maxVUs: 300,
    },
  },
  thresholds: {
    http_req_failed:   ['rate<0.01'],
    http_req_duration: ['p(95)<100', 'p(99)<300'],
  },
};

export default function () {
  const base = BASES[Math.floor(Math.random() * BASES.length)];
  const res = http.get(`${BASE_URL}/api/portfolio?base=${base}`);
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
