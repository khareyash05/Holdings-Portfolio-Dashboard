// cache test for forex and exchange data
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
