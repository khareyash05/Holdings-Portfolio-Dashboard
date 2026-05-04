import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = 'http://localhost:8080';
const RPS = 30;
const DURATION = '30s';

const ok = new Counter('rl_ok_200');
const limited = new Counter('rl_limited_429');

export const options = {
  scenarios: {
    ratelimit: {
      executor: 'constant-arrival-rate',
      rate: RPS,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 30,
      maxVUs: 100,
    },
  },
  thresholds: {
    rl_ok_200: ['count>0'],
    rl_limited_429: ['count>0'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/portfolio?base=INR`);
  check(res, { 'status is 200 or 429': (r) => r.status === 200 || r.status === 429 });

  if (res.status === 200) ok.add(1);
  else if (res.status === 429) limited.add(1);
}

export function handleSummary(data) {
  const oks = data.metrics.rl_ok_200.values.count;
  const lim = data.metrics.rl_limited_429.values.count;
  const total = oks + lim;
  const pct = (n) => ((n / total) * 100).toFixed(1) + '%';

  const out = `
rate-limit run: ${RPS} RPS for ${DURATION}
  total: ${total}
  200:   ${oks} (${pct(oks)})
  429:   ${lim} (${pct(lim)})
`;
  return { stdout: out };
}
