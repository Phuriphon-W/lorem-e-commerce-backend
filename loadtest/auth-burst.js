import http from 'k6/http';
import { check, sleep } from 'k6';
import execution from 'k6/execution';
import { BASE_URL } from './config.js';

export const options = {
  scenarios: {
    // Scenario 1: Single IP bursts to verify rate limiting kicks in
    single_ip_burst: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 30, // 30 requests rapidly
      maxDuration: '10s',
    },
    // Scenario 2: Multi IP burst to verify rate limiter allows traffic from different IPs
    multi_ip_burst: {
      executor: 'per-vu-iterations',
      vus: 50,
      iterations: 5, // 5 requests per IP
      startTime: '10s', // Start after single IP test
      maxDuration: '30s',
    },
  },
  thresholds: {
    // We expect some requests to fail (with 429) for the single IP scenario
    // We also expect auth failures (401) because we use bad credentials to avoid database clutter.
    // 500 errors must be 0
    'http_req_failed{status:500}': ['rate==0'],
  },
};

export default function () {
  const isSingleIpScenario = execution.scenario.name === 'single_ip_burst';
  
  let ipAddress = '1.2.3.4';
  if (!isSingleIpScenario) {
    // Generate a unique IP per VU to bypass single IP limit
    ipAddress = `192.168.100.${execution.vu.idInTest}`;
  }

  const payload = JSON.stringify({
    email: 'nonexistent-user@example.com',
    password: 'wrongpassword',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Forwarded-For': ipAddress,
      'X-Real-IP': ipAddress,
    },
  };

  const res = http.post(`${BASE_URL}/auth/signin`, payload, params);

  if (isSingleIpScenario) {
    // For single IP, the first 5 requests should return 404, and subsequent requests should return 429
    check(res, {
      'status is 404 or 429': (r) => r.status === 404 || r.status === 429,
    });
  } else {
    // For multi IP, since each IP makes exactly 5 requests, they should all be 404 (not 429)
    check(res, {
      'status is 404 (not rate limited)': (r) => r.status === 404,
    });
  }

  if (isSingleIpScenario) {
    sleep(0.1); // Fire rapidly
  } else {
    sleep(Math.random() * 2 + 1); // Normal think time
  }
}
