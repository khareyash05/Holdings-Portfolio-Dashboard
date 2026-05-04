// this is the sse load test to test holding N connnections to /api/portfolio/stream

import http from 'k6/http';
import { Counter, Trend } from 'k6/metrics';

const BASE_URL = 'http://localhost:8080';
const BASES = ['INR', 'USD', 'EUR', 'GBP', 'JPY'];

const VUS  = parseInt('100', 10); // 100 virtual users
const HOLD = parseInt('20', 10); // each hold for about 20s

const heldOk         = new Counter('sse_held_ok');
const failedConnect  = new Counter('sse_failed_connect');
const bytesPerStream = new Trend('sse_bytes_per_stream', false);
const ttfb           = new Trend('sse_ttfb_ms', true);

export const options = {
  scenarios: {
    sse: {
      executor: 'per-vu-iterations',
      vus: VUS,
      iterations: 1,
      maxDuration: `${HOLD + 10}s`,
    },
  },
  thresholds: {
    sse_failed_connect: ['count<2'],
    sse_ttfb_ms:        ['p(95)<500'],
  },
};

export default function () {
  const base = BASES[Math.floor(Math.random() * BASES.length)];
  const url = `${BASE_URL}/api/portfolio/stream?base=${base}&interval=1s`;

  const res = http.get(url, {
    timeout: `${HOLD}s`,
    headers: { Accept: 'text/event-stream' },
  });

  const timedOut = res.error_code === 1050;
  const gotHeaders = res.timings.waiting > 0;

  if (gotHeaders) {
    ttfb.add(res.timings.waiting);
  }

  // since sse connections hold the gateway we need to ensure there is a timeout, if yes the connection is stable(also ensuring headers are set)
  if (timedOut && gotHeaders) {
    heldOk.add(1);
    bytesPerStream.add((res.body || '').length);
  } else if (!timedOut && res.status !== 200) {
    failedConnect.add(1);
  }
}
